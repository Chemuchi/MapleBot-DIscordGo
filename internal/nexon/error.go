package nexon

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// errorResponse : 넥슨 API가 4xx/429 등에서 공통으로 내려주는 에러 포맷
// {"error": {"name": "...", "message": "..."}}
type errorResponse struct {
	Error struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"error"`
}

// APIError : 넥슨 API 호출 실패를 나타내는 에러.
// 상태 코드별로 사용자 친화적인 메시지를 만들 때 StatusCode를 활용합니다.
type APIError struct {
	StatusCode int
	Name       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.StatusCode, e.Name, e.Message)
}

// FriendlyMessage : 상태 코드별로 사용자에게 보여줄 한국어 메시지를 반환합니다.
// (discord 패키지가 이 메시지를 그대로 임베드에 넣어 쓸 수 있도록 함)
func (e *APIError) FriendlyMessage(context string) string {
	switch e.StatusCode {
	case http.StatusBadRequest:
		return "요청 형식이 올바르지 않습니다."
	case http.StatusNotFound:
		return fmt.Sprintf("'%s'을(를) 찾을 수 없습니다. 입력값을 다시 확인해주세요.", context)
	case http.StatusTooManyRequests:
		return "API 요청 한도를 초과했습니다. 잠시 후 다시 시도해주세요."
	default:
		return fmt.Sprintf("%s (%s)", e.Message, e.Name)
	}
}

// parseAPIError : 에러 응답 바디를 파싱해서 *APIError로 변환합니다.
func parseAPIError(statusCode int, body []byte) error {
	var resp errorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// 에러 바디 파싱조차 실패한 경우, 상태 코드만이라도 알려줌
		return &APIError{
			StatusCode: statusCode,
			Name:       "UNKNOWN_ERROR",
			Message:    fmt.Sprintf("알 수 없는 오류가 발생했습니다 (status: %d)", statusCode),
		}
	}
	return &APIError{
		StatusCode: statusCode,
		Name:       resp.Error.Name,
		Message:    resp.Error.Message,
	}
}
