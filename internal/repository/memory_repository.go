package repository

import (
	"context"
	"coupon-issuance-system/internal/model"
	"errors"
	"fmt"
	//"log"
	"sync"
	"time"

	"coupon-issuance-system/gen/coupon"
)

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

	model.NewCampaign(campaign).UpdateStatusIfNeeded() // 상태 업데이트 (lazy evaluation)

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
	mutex             sync.RWMutex                // 전체 데이터 뮤텍스
	campaignMutexes   map[string]*sync.Mutex      // 캠페인별 뮤텍스 맵
	campaignMutexLock sync.Mutex                  // 캠페인 뮤텍스 맵 보호
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

	// 캠페인별 뮤텍스 가져오기
	campaignMutex := r.getCampaignMutex(campaignID)
	campaignMutex.Lock()
	defer campaignMutex.Unlock()

	// 읽기 잠금으로 캠페인 정보 확인
	r.mutex.RLock()
	pbCampaign, exists := r.campaigns[campaignID]
	if !exists {
		r.mutex.RUnlock()
		return nil, false, fmt.Errorf("해당 캠페인이 존재하지 않습니다. id: %s", campaignID)
	}
	domainCampaign := model.NewCampaign(pbCampaign)

	// 쿠폰 발급 가능 여부 확인
	if !domainCampaign.CanIssueCoupon() {
		r.mutex.RUnlock()
		return nil, false, fmt.Errorf("해당 캠페인에서 쿠폰을 발급할 수 없습니다. id: %s", campaignID)
	}

	// 쓰기 잠금으로 실제 발급 처리
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 쿠폰 생성 및 저장
	if err := domainCampaign.IssueCoupon(); err != nil {
		if errors.Is(err, model.ErrSoldOut) || errors.Is(err, model.ErrNotActive) || errors.Is(err, model.ErrCannotIssue) {
			return nil, false, nil
		}
		return nil, false, err
	}
	newCoupon := &coupon.Coupon{
		CouponCode: couponCode,
		CampaignId: campaignID,
		IssuedAt:   time.Now().Unix(),
		IssuedTo:   userID,
	}
	r.coupons[campaignID] = append(r.coupons[campaignID], newCoupon)
	r.couponsByCode[couponCode] = newCoupon

	return newCoupon, true, nil
}

func (r *MemoryCouponRepository) getCampaignMutex(campaignID string) *sync.Mutex {
	r.campaignMutexLock.Lock()

	campaignMutex, exists := r.campaignMutexes[campaignID]
	if !exists {
		campaignMutex = &sync.Mutex{} // 없으면 새로 생성
		r.campaignMutexes[campaignID] = campaignMutex
	}

	r.campaignMutexLock.Unlock()
	return campaignMutex
}
