package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"coupon-issuance-system/gen/coupon"
	"coupon-issuance-system/gen/coupon/couponconnect"
)

func main() {
	fmt.Println("🚀 쿠폰 발급 부하테스트 시작")

	// 설정
	serverURL := "http://localhost:8080"
	workerCount := 100
	totalRequests := 1000
	couponLimit := 50

	fmt.Printf("설정: %d개 워커가 %d개 요청으로 %d개 쿠폰 발급 시도\n\n",
		workerCount, totalRequests, couponLimit)

	client := couponconnect.NewCouponServiceClient(http.DefaultClient, serverURL)
	ctx := context.Background()

	// 1. 캠페인 생성
	campaignID := createCampaign(ctx, client, couponLimit)

	// 2. 부하테스트 실행
	runLoadTest(ctx, client, campaignID, workerCount, totalRequests)

	// 3. 결과 확인
	checkResults(ctx, client, campaignID, couponLimit)
}

func createCampaign(ctx context.Context, client couponconnect.CouponServiceClient, limit int) string {
	fmt.Print("📋 캠페인 생성 중... ")

	req := connect.NewRequest(&coupon.CreateCampaignRequest{
		Name:          "부하테스트",
		StartTime:     time.Now().Add(1 * time.Second).Unix(),
		TotalQuantity: int32(limit),
	})

	resp, err := client.CreateCampaign(ctx, req)
	if err != nil || resp.Msg.Campaign == nil {
		log.Fatalf("캠페인 생성 실패: %v", err)
	}

	fmt.Printf("완료 (ID: %s)\n", resp.Msg.Campaign.CampaignId)

	// 캠페인 활성화 대기
	fmt.Print("⏳ 캠페인 활성화 대기... ")
	time.Sleep(2 * time.Second)
	fmt.Println("완료")

	return resp.Msg.Campaign.CampaignId
}

func runLoadTest(
	ctx context.Context,
	client couponconnect.CouponServiceClient,
	campaignID string,
	workerCount, totalRequests int,
) {
	fmt.Printf("🔥 %d개 워커로 %d개 요청 시작...\n", workerCount, totalRequests)

	var successCount int64
	var failCount int64
	var wg sync.WaitGroup

	// 작업 큐 생성
	workQueue := make(chan int, totalRequests) // 채널은 자바의 BlockingQueue 와 유사함 FIFO
	for i := 0; i < totalRequests; i++ {
		workQueue <- i // i 를 채널에 전송한다.
	}
	close(workQueue) // 전송을 중단 (더 이상 넣을 데이터 없음을 알림)

	start := time.Now()

	// 워커들 시작 : 작업큐의 작업을 워커풀 (workerCount)이 나눠서 처리하는 형태
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// 채널에서 하나씩 가져와서 처리 (채널이 빌 때까지 반복)
			for requestID := range workQueue {

				req := connect.NewRequest(&coupon.IssueCouponRequest{
					CampaignId: campaignID,
					UserId:     fmt.Sprintf("user-%d-%d", workerID, requestID),
				})

				resp, err := client.IssueCoupon(ctx, req)

				if err == nil && resp.Msg.Success {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failCount, 1)
				}

				// 진행률 출력 (100개마다)
				total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failCount)
				if total%100 == 0 {
					fmt.Printf("   진행: %d/%d (성공: %d)\n", total, totalRequests, successCount)
				}
			}

		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("\n✅ 부하테스트 완료!\n")
	fmt.Printf("   소요시간: %v\n", duration)
	fmt.Printf("   성공: %d개\n", successCount)
	fmt.Printf("   실패: %d개\n", failCount)
	fmt.Printf("   RPS: %.0f\n", float64(totalRequests)/duration.Seconds())
}

func checkResults(
	ctx context.Context,
	client couponconnect.CouponServiceClient,
	campaignID string,
	expectedLimit int) {

	req := connect.NewRequest(&coupon.GetCampaignRequest{
		CampaignId: campaignID,
	})

	resp, err := client.GetCampaign(ctx, req)
	if err != nil {
		fmt.Printf("실패: %v\n", err)
		return
	}

	campaign := resp.Msg.Campaign
	issuedCoupons := resp.Msg.IssuedCoupons

	fmt.Printf("결과 확인:\n")
	fmt.Printf("   발급된 쿠폰: %d개 (예상: %d개)\n", len(issuedCoupons), expectedLimit)
	fmt.Printf("   캠페인 상태: %s\n", campaign.Status)

	if int(campaign.IssuedQuantity) == len(issuedCoupons) {
		fmt.Printf("✅ 데이터 일관성 확인\n")
	} else {
		fmt.Printf("❌ 데이터 불일치: 캠페인의 IssuedQuantity (%d) vs issuedCoupons(%d)\n",
			campaign.IssuedQuantity, len(issuedCoupons))
	}
}
