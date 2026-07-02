// Package nexon은 넥슨 오픈 API(https://openapi.nexon.com) 호출을 담당합니다.
// 이 패키지는 디스코드에 대해 전혀 알지 못합니다 — 순수하게 "메이플 API 클라이언트"
// 역할만 하고, 결과를 어떻게 보여줄지는 discord 패키지가 결정합니다.
package nexon

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL        = "https://open.api.nexon.com"
	defaultTimeout = 10 * time.Second
)

// Client : 넥슨 오픈 API 호출용 클라이언트.
// 앞으로 캐릭터 스탯, 장비, 유니온 등 다른 API를 추가할 때도
// 이 Client의 메서드로 계속 확장하면 됩니다 (character.go, union.go 등에 분리).
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient : API 키를 받아 새 Client를 생성합니다.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// get : 넥슨 오픈 API에 GET 요청을 보내고, 성공 시 응답 바디를,
// 실패(비-200) 시 파싱된 *APIError를 반환하는 공통 함수.
// 새 엔드포인트를 추가할 때 이 함수를 재사용하면 됩니다.
func (c *Client) get(endpoint string, query url.Values) ([]byte, error) {
	reqURL := baseURL + endpoint
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	req.Header.Set("x-nxopen-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("요청 전송 실패: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, body)
	}

	return body, nil
}
