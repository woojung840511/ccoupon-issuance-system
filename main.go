package main

import (
	"log"
	"net/http"
	"time"

	"coupon-issuance-system/gen/coupon/couponconnect"
	"coupon-issuance-system/internal/handler"
	"coupon-issuance-system/internal/repository"
	"coupon-issuance-system/internal/service"
)

/*
# ConnectRPC:
- HTTP 위에서 동작하는 RPC 프레임워크
- HTTP 헤더로 프로토콜을 구분함
- ConnectRPC의 장점:
	- 서버 코드는 동일하게 유지
	- 클라이언트만 바꾸면 성능 향상 가능
	- 개발 시엔 JSON, 프로덕션에선 Protobuf 가능

# 쿠폰 발급 서버 구조
클라이언트 요청
    ↓
HTTP 서버 (mux.Handle)  	← 여기서 라우팅!
    ↓
ConnectRPC Handler      ← 여기서 프로토콜 구분!
    ↓
비즈니스 로직    			← CouponService

- HTTP 라우팅: "어떤 서비스로 갈지" 결정
- ConnectRPC: "어떤 프로토콜로 파싱할지" 결정
*/

func main() {
	// 의존성 주입
	campaignRepo := repository.NewMemoryCampaignRepository()
	couponRepo := repository.NewMemoryCouponRepository(campaignRepo)
	codeGenerator := service.NewCouponCodeGenerator()
	couponService := service.NewCouponService(campaignRepo, couponRepo, codeGenerator)

	// ConnectRPC 핸들러 등록
	couponHandler := handler.NewCouponServiceHandler(couponService)
	path, httpHandler := couponconnect.NewCouponServiceHandler(couponHandler)

	// HTTP 라우팅
	mux := http.NewServeMux()     // ServeMux = HTTP 라우터 (Spring의 @RequestMapping 같은 역할)
	mux.Handle(path, httpHandler) // ServeMux는 여러 URL 경로를 각각 다른 핸들러로 분배하는 라우터 역할을 함

	// 미들웨어 추가
	finalHandler := corsMiddleware(loggingMiddleware(mux))

	// 서버 설정
	/*
		server := &http.Server{
			Addr:    ":8080",
			Handler: h2c.NewHandler(finalHandler, &http2.Server{}),
		}
	*/
	server := &http.Server{
		Addr:    ":8080",
		Handler: finalHandler, // h2c 제거
	}

	log.Printf("🚀 쿠폰 발급 서버 시작: http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}

// CORS 미들웨어
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r) // Go에서 HTTP 요청을 처리하는 표준 인터페이스
	})
}

// 로깅 미들웨어
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 요청 로깅
		log.Printf("📥 [%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)

		// 응답 시간 로깅
		duration := time.Since(start)
		log.Printf("📤 [%s] %s - %v", r.Method, r.URL.Path, duration)
	})
}
