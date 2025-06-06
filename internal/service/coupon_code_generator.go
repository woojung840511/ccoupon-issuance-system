package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"unicode/utf8"
)

// CouponCodeGenerator 쿠폰 코드 생성기
type CouponCodeGenerator struct {
	koreanChars []rune
	numberChars []rune
}

// NewCouponCodeGenerator 생성자
func NewCouponCodeGenerator() *CouponCodeGenerator {
	return &CouponCodeGenerator{
		koreanChars: []rune{'가', '나', '다', '라', '마', '바', '사', '아', '자', '차', '카', '타', '파', '하'},
		numberChars: []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
	}
}

// GenerateCode 쿠폰 코드 생성 (캠페인명 기반 + 랜덤)
func (g *CouponCodeGenerator) GenerateCode(campaignName string) (string, error) {

	prefix := g.extractPrefix(campaignName)

	remainingLength := 10 - utf8.RuneCountInString(prefix) // 7~8 자 랜덤 부분 생성

	randomPart, err := g.generateRandomPart(remainingLength)
	if err != nil {
		return "", err
	}

	return prefix + randomPart, nil
}

// extractPrefix 캠페인명에서 의미있는 prefix 추출. 길이 2-3자
func (g *CouponCodeGenerator) extractPrefix(campaignName string) string {
	if campaignName == "" {
		return "쿠폰"
	}

	var hangulRunes []rune
	for _, r := range campaignName {
		if r >= '가' && r <= '힣' {
			hangulRunes = append(hangulRunes, r)
		}
	}

	if len(hangulRunes) == 0 {
		return "쿠폰"
	}

	if len(hangulRunes) >= 3 {
		return string(hangulRunes[:3])
	} else if len(hangulRunes) >= 2 {
		return string(hangulRunes[:2])
	} else {
		return string(hangulRunes[0]) + "폰"
	}
}

// generateRandomPart 랜덤 부분 생성 (한글 + 숫자 혼합)
func (g *CouponCodeGenerator) generateRandomPart(length int) (string, error) {
	var result []rune

	// 앞쪽 2자는 한글로
	hangulCount := 2

	// 한글 부분
	for i := 0; i < hangulCount; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(g.koreanChars))))
		if err != nil {
			return "", err
		}
		result = append(result, g.koreanChars[idx.Int64()])
	}

	// 숫자 부분
	for i := hangulCount; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(g.numberChars))))
		if err != nil {
			return "", err
		}
		result = append(result, g.numberChars[idx.Int64()])
	}

	return string(result), nil
}

// GenerateUniqueCode 중복되지 않는 쿠폰 코드 생성
func (g *CouponCodeGenerator) GenerateUniqueCode(campaignName string, checkDuplicate func(string) bool) (string, error) {
	maxRetries := 100

	for i := 0; i < maxRetries; i++ {
		code, err := g.GenerateCode(campaignName)
		if err != nil {
			return "", err
		}

		// 중복 검사
		if !checkDuplicate(code) {
			return code, nil
		}
	}

	return "", fmt.Errorf("쿠폰 코드 중복 방지를 위한 최대 시도 횟수(%d) 초과", maxRetries)
}
