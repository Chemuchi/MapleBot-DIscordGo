// Package format은 숫자/날짜 등을 사용자에게 보여줄 형태로 바꾸는
// 순수 유틸 함수들을 모아둡니다. 디스코드나 넥슨 API를 전혀 몰라도 되는 코드만 둡니다.
package format

import (
	"fmt"
	"time"
)

// Comma : 정수를 천 단위 콤마가 포함된 문자열로 변환합니다. (예: 1234567 -> "1,234,567")
func Comma(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var result []byte
	for i, c := range []byte(s) {
		if i != 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	if neg {
		return "-" + string(result)
	}
	return string(result)
}

// Date : 넥슨 API가 내려주는 날짜 문자열을 "2026년 01월 02일" 형태로 변환합니다.
// 넥슨 API는 초 단위를 포함(RFC3339)하거나 생략한 형태("...T00:00+09:00") 둘 다
// 내려줄 수 있어서 여러 레이아웃을 순서대로 시도합니다.
func Date(raw string) string {
	layouts := []string{
		time.RFC3339,             // 2026-06-18T00:00:00+09:00
		"2006-01-02T15:04Z07:00", // 2026-06-18T00:00+09:00 (초 생략)
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006년 01월 02일")
		}
	}

	// 모든 레이아웃 파싱 실패 시 원본 문자열 그대로 반환
	return raw
}

// DateTime : 넥슨 API가 내려주는 날짜 문자열을 "2006년 01월 02일 15시 04분" 형태로 변환합니다.
func DateTime(raw string) string {
	layouts := []string{
		time.RFC3339,             // 2026-06-18T00:00:00+09:00
		"2006-01-02T15:04Z07:00", // 2026-06-18T00:00+09:00 (초 생략)
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006년 01월 02일 15시 04분")
		}
	}

	return raw
}

