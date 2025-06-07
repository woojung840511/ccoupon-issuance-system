// internal/service/validation.go
package service

import (
	"time"

	"coupon-issuance-system/gen/coupon"
)

// ValidationResult 검증 결과
type ValidationResult struct {
	IsValid bool
	Message string
}

// Valid 검증 성공
func Valid() ValidationResult {
	return ValidationResult{IsValid: true}
}

// Invalid 검증 실패
func Invalid(message string) ValidationResult {
	return ValidationResult{IsValid: false, Message: message}
}

// validateCreateCampaignRequest 캠페인 생성 요청 검증
func validateCreateCampaignRequest(req *coupon.CreateCampaignRequest) ValidationResult {
	if req.Name == "" {
		return Invalid("캠페인 이름은 필수입니다")
	}

	if req.TotalQuantity <= 0 {
		return Invalid("발급 수량은 1개 이상이어야 합니다")
	}

	now := time.Now().Unix()
	if req.StartTime < now {
		return Invalid("시작 시간은 현재 시간 이후여야 합니다")
	}

	return Valid()
}

// validateIssueCouponRequest 쿠폰 발급 요청 검증
func validateIssueCouponRequest(req *coupon.IssueCouponRequest) ValidationResult {
	if req.CampaignId == "" {
		return Invalid("캠페인 ID는 필수입니다")
	}

	if req.UserId == "" {
		return Invalid("사용자 ID는 필수입니다")
	}

	return Valid()
}

// validateGetCampaignRequest 캠페인 조회 요청 검증
func validateGetCampaignRequest(req *coupon.GetCampaignRequest) ValidationResult {
	if req.CampaignId == "" {
		return Invalid("캠페인 ID는 필수입니다")
	}

	return Valid()
}
