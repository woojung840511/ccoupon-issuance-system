package handler

import (
	"connectrpc.com/connect"
	"context"
	"coupon-issuance-system/gen/coupon"
	"coupon-issuance-system/gen/coupon/couponconnect"
	"coupon-issuance-system/internal/service"
	"log"
)

type CouponServiceHandler struct {
	service *service.CouponService
}

func NewCouponServiceHandler(service *service.CouponService) *CouponServiceHandler {
	return &CouponServiceHandler{
		service: service,
	}
}

// CreateCampaign implements couponconnect.CouponServiceHandler
func (h *CouponServiceHandler) CreateCampaign(
	ctx context.Context,
	req *connect.Request[coupon.CreateCampaignRequest], // 래핑된 요청. 제네릭
) (*connect.Response[coupon.CreateCampaignResponse], // 래핑된 응답
	error) {

	// req.Msg 로 실제 데이터에 접근함
	log.Printf("CreateCampaign 요청: %+v", req.Msg)

	response, err := h.service.CreateCampaign(ctx, req.Msg)
	if err != nil {
		log.Printf("CreateCampaign 처리 중 오류: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	log.Printf("CreateCampaign 응답: %+v", response)
	return connect.NewResponse(response), nil
}

func (h *CouponServiceHandler) GetCampaign(
	ctx context.Context,
	req *connect.Request[coupon.GetCampaignRequest],
) (*connect.Response[coupon.GetCampaignResponse], error) {

	log.Printf("GetCampaign 요청: %+v", req.Msg)

	response, err := h.service.GetCampaign(ctx, req.Msg)
	if err != nil {
		log.Printf("GetCampaign 처리 중 오류: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	log.Printf("GetCampaign 응답: 캠페인=%s, 발급된쿠폰수=%d",
		response.Campaign.Name, len(response.IssuedCoupons))

	return connect.NewResponse(response), nil
}

func (h *CouponServiceHandler) IssueCoupon(
	ctx context.Context,
	req *connect.Request[coupon.IssueCouponRequest],
) (*connect.Response[coupon.IssueCouponResponse], error) {

	log.Printf("IssueCoupon 요청: CampaignID=%s, UserID=%s",
		req.Msg.CampaignId, req.Msg.UserId)

	response, err := h.service.IssueCoupon(ctx, req.Msg)
	if err != nil {
		log.Printf("IssueCoupon 처리 중 오류: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if response.Success {
		log.Printf("IssueCoupon 성공: UserID=%s, CouponCode=%s",
			req.Msg.UserId, response.Coupon.CouponCode)
	} else {
		log.Printf("IssueCoupon 실패: UserID=%s, 이유=%s",
			req.Msg.UserId, response.Message)
	}

	return connect.NewResponse(response), nil
}

// Go의 컴파일 타임 인터페이스 검증
var _ couponconnect.CouponServiceHandler = (*CouponServiceHandler)(nil) // nil을 *CouponServiceHandler 타입으로 캐스팅
// 컴파일 확인해보기 go build ./...
