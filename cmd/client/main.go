package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"coupon-issuance-system/gen/coupon"
	"coupon-issuance-system/gen/coupon/couponconnect"
)

func main() {
	client := couponconnect.NewCouponServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)
	ctx := context.Background()

	// 1. 캠페인 생성
	fmt.Print("📋 2초 뒤에 시작하는 테스트 캠페인 생성 중... ")
	startTime := time.Now().Add(2 * time.Second).Unix()

	createReq := connect.NewRequest(&coupon.CreateCampaignRequest{
		Name:          "데모 캠페인",
		StartTime:     startTime,
		TotalQuantity: 3,
	})

	createResp, err := client.CreateCampaign(ctx, createReq)
	if err != nil {
		fmt.Printf("❌ 데모 켐페인 생성 실패: %v\n", err)
		return
	}
	campaignID := createResp.Msg.Campaign.CampaignId
	fmt.Printf("✅ 캠페인 생성 완료 (ID: %s)\n", campaignID)

	// 2. 활성화 대기
	fmt.Print("📋 캠페인 시작시간 대기 중... ")
	time.Sleep(3 * time.Second)
	fmt.Println("✅ 완료")

	// 3. 쿠폰 발급
	fmt.Print("📋 쿠폰 발급 중... ")
	issueReq := connect.NewRequest(&coupon.IssueCouponRequest{
		CampaignId: campaignID,
		UserId:     "demo-user",
	})

	issueResp, err := client.IssueCoupon(ctx, issueReq)
	if err != nil {
		fmt.Printf("❌ 쿠폰 발급 실패: %v\n", err)
		return
	}

	if issueResp.Msg.Success {
		fmt.Printf("✅ 쿠폰 발급 완료 (쿠폰코드: %s)\n", issueResp.Msg.Coupon.CouponCode)
	} else {
		fmt.Printf("❌ 쿠폰 발급 실패: %s\n", issueResp.Msg.Message)
	}

	// 4. 최종 상태 확인
	fmt.Print("📋 최종 상태 확인 중... ")
	getReq := connect.NewRequest(&coupon.GetCampaignRequest{
		CampaignId: campaignID,
	})

	getResp, err := client.GetCampaign(ctx, getReq)
	if err != nil {
		fmt.Printf("❌ 캠페인 조회 실패: %v\n", err)
		return
	}

	campaign := getResp.Msg.Campaign
	coupons := getResp.Msg.IssuedCoupons
	if len(coupons) != int(campaign.IssuedQuantity) {
		fmt.Printf("❌ 발급된 쿠폰 수 불일치: 예상 %d, 실제 %d\n", campaign.IssuedQuantity, len(coupons))
		return
	}
	fmt.Printf("✅ 완료 (발급: %d/%d개)\n", campaign.IssuedQuantity, campaign.TotalQuantity)

	fmt.Println("데모 완료")
}
