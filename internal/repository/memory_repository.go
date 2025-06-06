package repository

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"coupon-issuance-system/gen/coupon"
)

func updateCampaignStatus(campaign *coupon.Campaign) {
	now := time.Now().Unix()

	if campaign.Status == coupon.CampaignStatus_WAITING && now >= campaign.StartTime {
		campaign.Status = coupon.CampaignStatus_ACTIVE
		log.Printf("캠페인 %s 자동 시작됨", campaign.CampaignId)

	} else if campaign.Status == coupon.CampaignStatus_ACTIVE && campaign.IssuedQuantity >= campaign.TotalQuantity {
		campaign.Status = coupon.CampaignStatus_COMPLETED
		log.Printf("캠페인 %s 자동 완료됨", campaign.CampaignId)
	}
}

type MemoryCampaignRepository struct {
	campaigns map[string]*coupon.Campaign
	mutex     sync.RWMutex
}

func NewMemoryCampaignRepository() *MemoryCampaignRepository {
	return &MemoryCampaignRepository{
		campaigns: make(map[string]*coupon.Campaign),
	}
}

func (r *MemoryCampaignRepository) Save(ctx context.Context, campaign *coupon.Campaign) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.campaigns[campaign.CampaignId] = campaign
	return nil
}

func (r *MemoryCampaignRepository) GetByID(ctx context.Context, id string) (*coupon.Campaign, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	campaign, exists := r.campaigns[id]
	if !exists {
		return nil, fmt.Errorf("해당 캠페인이 존재하지 않습니다. id: %s", id)
	}

	updateCampaignStatus(campaign) // 효율을 위해 Lazy 하게 평가

	return campaign, nil
}

func (r *MemoryCampaignRepository) Update(ctx context.Context, campaign *coupon.Campaign) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.campaigns[campaign.CampaignId]; !exists {
		return fmt.Errorf("해당 캠페인이 존재하지 않습니다. id: %s", campaign.CampaignId)
	}

	r.campaigns[campaign.CampaignId] = campaign
	return nil
}

func (r *MemoryCampaignRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.campaigns[id]; !exists {
		return fmt.Errorf("해당 캠페인이 존재하지 않습니다. id: %s", id)
	}

	delete(r.campaigns, id)
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////

type MemoryCouponRepository struct {
	coupons           map[string][]*coupon.Coupon // campaignID -> coupons
	couponsByCode     map[string]*coupon.Coupon   // couponCode -> coupon , 중복이지만 인덱싱 기능
	campaigns         map[string]*coupon.Campaign // campaignRepo.campaigns
	mutex             sync.RWMutex
	campaignMutexes   map[string]*sync.Mutex
	campaignMutexLock sync.Mutex
}

func NewMemoryCouponRepository(campaignRepo *MemoryCampaignRepository) *MemoryCouponRepository {
	return &MemoryCouponRepository{
		coupons:         make(map[string][]*coupon.Coupon),
		couponsByCode:   make(map[string]*coupon.Coupon),
		campaigns:       campaignRepo.campaigns,
		campaignMutexes: make(map[string]*sync.Mutex),
	}
}

func (r *MemoryCouponRepository) Save(ctx context.Context, coupon *coupon.Coupon) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.coupons[coupon.CampaignId] = append(r.coupons[coupon.CampaignId], coupon)
	r.couponsByCode[coupon.CouponCode] = coupon

	return nil
}

func (r *MemoryCouponRepository) GetByCampaignID(ctx context.Context, campaignID string) ([]*coupon.Coupon, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	coupons, exists := r.coupons[campaignID]
	if !exists {
		return []*coupon.Coupon{}, nil
	}

	return coupons, nil
}

func (r *MemoryCouponRepository) GetByCode(ctx context.Context, code string) (*coupon.Coupon, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	cp, exists := r.couponsByCode[code]
	if !exists {
		return nil, fmt.Errorf("해당 쿠폰이 존재하지 않습니다. code: %s", code)
	}

	return cp, nil
}

func (r *MemoryCouponRepository) IssueCouponAtomic(ctx context.Context, campaignID, userID, couponCode string) (*coupon.Coupon, bool, error) {

	// 1. 캠페인별 뮤텍스 가져오기, 없으면 생성
	r.campaignMutexLock.Lock()
	campaignMutex, exists := r.campaignMutexes[campaignID]
	if !exists {
		campaignMutex = &sync.Mutex{}
		r.campaignMutexes[campaignID] = campaignMutex
	}
	r.campaignMutexLock.Unlock()

	// 2. 해당 캠페인에 대한 독점적 접근
	campaignMutex.Lock()
	defer campaignMutex.Unlock()

	// 3. 읽기 잠금으로 캠페인 정보 확인
	r.mutex.RLock()
	campaign, exists := r.campaigns[campaignID]
	if !exists {
		r.mutex.RUnlock()
		return nil, false, fmt.Errorf("해당 캠페인이 존재하지 않습니다. id: %s", campaignID)
	}

	// 캠페인 상태 업데이트 // todo -> DDD 로 변경해보기
	updateCampaignStatus(campaign) // lazy evaluation..

	// 4. 발급 가능 여부 확인 // todo -> DDD 로 변경해보기
	if campaign.IssuedQuantity >= campaign.TotalQuantity {
		r.mutex.RUnlock()
		return nil, false, nil // 품절
	}

	if campaign.Status != coupon.CampaignStatus_ACTIVE {
		r.mutex.RUnlock()
		return nil, false, nil // 캠페인이 활성 상태가 아님
	}
	r.mutex.RUnlock()

	// 5. 쓰기 잠금으로 실제 발급 처리
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 다시 한 번 확인
	// 여기서 확인하지 않았을 경우 일어날 수 있는 위험 :
	// 발급 가능여부까지 확인하고 쓰기 락을 대기하던 스레드가 다시 락을 획득한 시점에
	// 다른 스레드가 발급 가능 수량을 모두 소진한 상황임을 인지하지 못할 수 있음
	campaign = r.campaigns[campaignID]
	if campaign.IssuedQuantity >= campaign.TotalQuantity {
		return nil, false, nil // 품절
	}

	// 6. 쿠폰 생성 및 저장 // todo -> DDD 로 변경해보기
	newCoupon := &coupon.Coupon{
		CouponCode: couponCode,
		CampaignId: campaignID,
		IssuedAt:   time.Now().Unix(),
		IssuedTo:   userID,
	}

	// 7. 원자적 업데이트
	r.coupons[campaignID] = append(r.coupons[campaignID], newCoupon)
	r.couponsByCode[couponCode] = newCoupon
	campaign.IssuedQuantity++

	// 8. 캠페인 완료 상태 확인
	if campaign.IssuedQuantity >= campaign.TotalQuantity {
		campaign.Status = coupon.CampaignStatus_COMPLETED
	}

	return newCoupon, true, nil
}
