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
		wg.Add(1) // 고루틴 수 증가
		go func(index int) {
			defer wg.Done() // 고루틴 완료 시 wg.Done() 호출

			userID := fmt.Sprintf("user-%d", index)
			couponCode := fmt.Sprintf("CODE%d", index)

			_, success, _ := couponRepo.IssueCouponAtomic(ctx, "t2", userID, couponCode)

			if success {
				mu.Lock() // 다른 고루틴 대기
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait() // 모든 고루틴이 완료될 때까지 대기

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
