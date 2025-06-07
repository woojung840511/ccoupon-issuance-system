package model

import (
	pb "coupon-issuance-system/gen/coupon"
	"errors"
	"log"
	"time"
)

type Campaign struct {
	*pb.Campaign // 임베딩할 때 필드명을 명시하지 않으면, Go는 타입명을 필드명으로 자동 사용
}

var (
	ErrSoldOut     = errors.New("SOLD_OUT")
	ErrNotActive   = errors.New("CAMPAIGN_NOT_ACTIVE")
	ErrCannotIssue = errors.New("CANNOT_ISSUE_COUPON")
)

func NewCampaign(pbCampaign *pb.Campaign) *Campaign {
	return &Campaign{Campaign: pbCampaign}
}

func (c *Campaign) CanIssueCoupon() (bool, string) {
	c.UpdateStatusIfNeeded()

	switch c.Status {
	case pb.CampaignStatus_UNSPECIFIED:
		return false, "캠페인이 아직 시작되지 않았습니다"

	case pb.CampaignStatus_WAITING:
		return false, "캠페인이 아직 활성상태가 아닙니다"

	case pb.CampaignStatus_ACTIVE:
		if c.IssuedQuantity >= c.TotalQuantity {
			return false, "쿠폰이 모두 소진되었습니다"
		}

	case pb.CampaignStatus_COMPLETED:
		return false, "캠페인이 종료되었습니다"
	}

	return true, ""
}

func (c *Campaign) UpdateStatusIfNeeded() {
	now := time.Now().Unix()

	if c.Status == pb.CampaignStatus_WAITING && now >= c.StartTime {

		c.Status = pb.CampaignStatus_ACTIVE
		log.Printf("Campaign status 변경. before : %s, after : %s\n", pb.CampaignStatus_WAITING, c.Status)

	} else if c.Status == pb.CampaignStatus_ACTIVE && c.IssuedQuantity >= c.TotalQuantity {

		c.Status = pb.CampaignStatus_COMPLETED
		log.Printf("Campaign status 변경. before : %s, after : %s\n", pb.CampaignStatus_ACTIVE, c.Status)

	}
}

func (c *Campaign) IssueCoupon() (bool, string) {
	canIssue, failMsg := c.CanIssueCoupon()
	if !canIssue {
		return false, failMsg
	}

	c.IssuedQuantity++
	log.Printf("쿠폰이 발급되었습니다. 현재 발급된 쿠폰 수량: %d", c.IssuedQuantity)

	c.UpdateStatusIfNeeded()
	return true, ""
}
