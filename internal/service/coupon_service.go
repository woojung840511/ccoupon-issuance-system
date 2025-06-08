package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"coupon-issuance-system/gen/coupon"
	"coupon-issuance-system/internal/repository"
)

type CouponService struct {
	campaignRepo *repository.MemoryCampaignRepository
	couponRepo   *repository.MemoryCouponRepository
	codeGen      *CouponCodeGenerator
}

func NewCouponService(
	campaignRepo *repository.MemoryCampaignRepository,
	couponRepo *repository.MemoryCouponRepository,
	codeGenerator *CouponCodeGenerator,
) *CouponService {
	return &CouponService{
		campaignRepo: campaignRepo,
		couponRepo:   couponRepo,
		codeGen:      codeGenerator,
	}
}

func (s *CouponService) CreateCampaign(
	ctx context.Context,
	req *coupon.CreateCampaignRequest,
) (*coupon.CreateCampaignResponse, error) {
	// 입력 검증
	validation := validateCreateCampaignRequest(req)
	if !validation.IsValid {
		return &coupon.CreateCampaignResponse{
			Message: validation.Message,
		}, nil
	}

	now := time.Now().Unix()

	campaignID := fmt.Sprintf("campaign_%d", time.Now().UnixNano()) // 나노초 단위

	// 캠페인 상태 결정하기
	status := coupon.CampaignStatus_WAITING
	if req.StartTime <= now {
		status = coupon.CampaignStatus_ACTIVE
	}

	campaign := &coupon.Campaign{
		CampaignId:     campaignID,
		Name:           req.Name,
		StartTime:      req.StartTime,
		TotalQuantity:  req.TotalQuantity,
		IssuedQuantity: 0,
		Status:         status,
		CreatedAt:      now,
	}

	err := s.campaignRepo.Save(ctx, campaign)
	if err != nil {
		log.Printf("캠페인 저장 실패: %v", err)
		return &coupon.CreateCampaignResponse{
			Message: "캠페인 생성에 실패했습니다",
		}, err
	}

	log.Printf("캠페인이 생성되었습니다. ID: %s, 이름: %s", campaignID, req.Name)

	return &coupon.CreateCampaignResponse{
		Campaign: campaign,
		Message:  "캠페인이 성공적으로 생성되었습니다",
	}, nil
}

func (s *CouponService) GetCampaign(
	ctx context.Context,
	req *coupon.GetCampaignRequest,
) (*coupon.GetCampaignResponse, error) {
	// 입력 검증
	validation := validateGetCampaignRequest(req)
	if !validation.IsValid {
		return &coupon.GetCampaignResponse{
			Message: validation.Message,
		}, nil
	}

	// 캠페인 조회
	campaign, err := s.campaignRepo.GetByID(ctx, req.CampaignId)
	if err != nil {
		log.Printf("캠페인 조회 실패: %v", err)
		return &coupon.GetCampaignResponse{
			Message: "캠페인을 찾을 수 없습니다",
		}, nil
	}

	// 발급된 쿠폰들 조회
	issuedCoupons, err := s.couponRepo.GetByCampaignID(ctx, req.CampaignId)
	if err != nil {
		log.Printf("쿠폰 조회 실패: %v", err)
		return &coupon.GetCampaignResponse{
			Campaign: campaign,
			Message:  "쿠폰 정보 조회에 실패했습니다",
		}, nil
	}

	return &coupon.GetCampaignResponse{
		Campaign:      campaign,
		IssuedCoupons: issuedCoupons,
		Message:       "조회 성공",
	}, nil
}

func (s *CouponService) IssueCoupon(
	ctx context.Context,
	req *coupon.IssueCouponRequest,
) (*coupon.IssueCouponResponse, error) {

	// 입력 검증
	validation := validateIssueCouponRequest(req)
	if !validation.IsValid {
		return &coupon.IssueCouponResponse{
			Success: false,
			Message: validation.Message,
		}, nil
	}

	// 쿠폰 코드 생성
	couponCode, err := s.generateUniqueCouponCode(ctx, req.CampaignId)
	if err != nil {
		log.Printf("쿠폰 코드 생성 실패: %v", err)
		return &coupon.IssueCouponResponse{
			Success: false,
			Message: "쿠폰 코드 생성에 실패했습니다",
		}, err
	}

	// 쿠폰 발급
	issuedCoupon, failMsg, err := s.couponRepo.IssueCoupon(ctx, req.CampaignId, req.UserId, couponCode)
	if err != nil {
		log.Printf("쿠폰 발급 처리 실패: %v", err)
		return &coupon.IssueCouponResponse{
			Success: false,
			Message: "쿠폰 발급 처리 중 오류가 발생했습니다",
		}, err
	}

	if issuedCoupon != nil {
		return &coupon.IssueCouponResponse{
			Success: false,
			Message: failMsg,
		}, err
	}

	// 성공
	log.Printf("쿠폰 발급 성공. 사용자: %s, 캠페인: %s, 쿠폰코드: %s",
		req.UserId, req.CampaignId, couponCode)

	return &coupon.IssueCouponResponse{
		Success: true,
		Coupon:  issuedCoupon,
		Message: "쿠폰이 성공적으로 발급되었습니다",
	}, nil
}

func (s *CouponService) generateUniqueCouponCode(
	ctx context.Context,
	campaignID string,
) (string, error) {

	// 캠페인 조회
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return "", fmt.Errorf("캠페인 조회 실패: %w", err)
	}

	// 중복 검사 함수 정의
	checkDuplicate := func(code string) bool {
		_, err := s.couponRepo.GetByCode(ctx, code)
		return err == nil // 에러가 없으면 중복 (이미 존재하기 때문)
	}

	couponCode, err := s.codeGen.GenerateUniqueCode(campaign.Name, checkDuplicate)
	if err != nil {
		return "", fmt.Errorf("쿠폰 코드 생성 실패: %w", err)
	}

	return couponCode, nil
}
