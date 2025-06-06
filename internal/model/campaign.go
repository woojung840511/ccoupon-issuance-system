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

func (c *Campaign) CanIssueCoupon() bool {
	c.UpdateStatusIfNeeded()
	return c.Status == pb.CampaignStatus_ACTIVE && c.IssuedQuantity < c.TotalQuantity
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

func (c *Campaign) IssueCoupon() error {
	if !c.CanIssueCoupon() { // 더블 체크
		if c.IssuedQuantity >= c.TotalQuantity {
			return ErrSoldOut
		}
		if c.Status != pb.CampaignStatus_ACTIVE {
			return ErrNotActive
		}
		return ErrCannotIssue
	}

	c.IssuedQuantity++
	log.Printf("쿠폰이 발급되었습니다. 현재 발급된 쿠폰 수량: %d", c.IssuedQuantity)

	c.UpdateStatusIfNeeded()
	return nil
}
