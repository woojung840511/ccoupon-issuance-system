package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"coupon-issuance-system/gen/coupon"
)

// 캠페인 저장
func TestBasicCampaignOperations(t *testing.T) {
	repo := NewMemoryCampaignRepository()
	ctx := context.Background()

	campaign := &coupon.Campaign{
		CampaignId:     "t1",
		Name:           "테스트",
		TotalQuantity:  10,
		IssuedQuantity: 0,
		Status:         coupon.CampaignStatus_WAITING,
	}

	repo.Save(ctx, campaign)

	saved, err := repo.GetByID(ctx, "t1")
	if err != nil || saved.Name != "테스트" {
		t.Fatal("t1 저장/조회 실패")
	}
}

func TestAtomicCouponIssue(t *testing.T) {
	campaignRepo := NewMemoryCampaignRepository()
	couponRepo := NewMemoryCouponRepository(campaignRepo)
	ctx := context.Background()

	// 쿠폰 5개 발급 가능
	campaign := &coupon.Campaign{
		CampaignId:     "t2",
		TotalQuantity:  5,
		IssuedQuantity: 0,
		Status:         coupon.CampaignStatus_ACTIVE,
		StartTime:      time.Now().Unix(),
	}
	campaignRepo.Save(ctx, campaign)

	// 20개 요청을 동시에 보내기
	numRequests := 20
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numRequests; i++ {
		wg.Add(1) // 내부 카운터를 1 증가 (대기할 고루틴 수 +1)
		go func(index int) {
			/*
				defer를 사용하는 이유 메모: 함수가 정상 종료되든, 에러로 종료되든 반드시 wg.Done()이 호출되도록 보장
			*/
			defer wg.Done() // 내부 카운터를 1 감소 (완료된 고루틴 수 +1)

			userID := fmt.Sprintf("user-%d", index)
			couponCode := fmt.Sprintf("CODE%d", index)

			_, success, _ := couponRepo.IssueCoupon(ctx, "t2", userID, couponCode)

			if success {
				mu.Lock() // 다른 고루틴 대기
				successCount++
				mu.Unlock()
			}
		}(i) // 각 고루틴마다 i의 복사본을 전달받아 독립적인 값을 가짐

		/*
			(i)의 역할 - 클로저 문제 해결
			각 고루틴이 독립적으로 index 값을 갖도록 보장
			이렇게 하지 않으면 모든 고루틴이 동일한 i 값을 참조하게 되어 마지막 값인 19만 사용하게 됨
		*/
	}

	wg.Wait() // 카운터가 0이 될 때까지 대기

	// 성공한 쿠폰 수 확인
	if successCount != 5 {
		t.Errorf("예상: 5개 성공, 실제: %d개 성공", successCount)
	}

	// 캠페인 발급된 쿠폰 수 확인
	finalCampaign, _ := campaignRepo.GetByID(ctx, "t2")
	if finalCampaign.IssuedQuantity != 5 {
		t.Errorf("캠페인 발급 수량이 잘못됨: %d", finalCampaign.IssuedQuantity)
	}

	t.Logf("동시성 테스트 통과: %d개 요청 중 %d개 성공", numRequests, successCount)
}
