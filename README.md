# 쿠폰 발급 시스템 (Coupon Issuance System)

## 과제 개요
지정된 시간에 제한된 수량의 쿠폰을 선착순으로 발급하는 시스템을 구현하는 과제입니다.
동시성 제어와 성능 최적화 문제를 다루며, 실제 서비스에서 발생할 수 있는 상황들을 고려한 설계가 필요합니다.

## 요구사항 분석
### 기능 요구사항
- RPC service and methods:
  - CreateCampaign: Create a new coupon campaign
  - GetCampaign: Get campaign information including all issued coupon codes
    (only include successfully issued ones)
  - IssueCoupon: Request coupon issuance on specific campaign
- requirements:
  - Issue exactly the specified number of coupons per campaign (no excess issuance)
  - Coupon issuance must automatically start at the exact specified date and time
  - Data consistency must be guaranteed throughout the issuance process
  - Each coupon must have a unique code across all campaigns (up to 10
    characters, consisting of Korean characters and numbers).
### 도전과제
- Implement a concurrency control mechanism to solve data consistency issues
  under high traffic conditions (500-1,000 requests per second).
- Implement horizontally scalable system (Scale-out)
- Explore and design solutions for various edge cases that might occur.
- Implement testing tools or scripts that can verify concurrency issues.

## 시스템 설계

### 데이터 모델

#### 쿠폰 캠페인 (Campaign)
```
- 캠페인 ID: 고유 식별자
- 캠페인 이름: 캠페인 명칭
- 시작 시간: 쿠폰 발급이 허용되는 시점
- 총 발급 수량: 발급 가능한 전체 쿠폰 개수
- 현재 발급 수량: 실제 발급된 쿠폰 개수
- 상태: 대기/진행중/완료
- 생성 시간: 캠페인 생성 시점
```

#### 쿠폰 (Coupon)
```
- 쿠폰 코드: 고유 식별 코드 (최대 10자)
- 캠페인 ID: 소속 캠페인
- 발급 시간: 쿠폰이 발급된 시점
- 발급 대상: 쿠폰을 받은 요청자 정보
```
