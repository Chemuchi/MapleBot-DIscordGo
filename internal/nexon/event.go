package nexon

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// EventNotice : /maplestory/v1/notice-event 응답의 개별 이벤트 항목
type EventNotice struct {
	Title          string `json:"title"`
	URL            string `json:"url"`
	ThumbnailURL   string `json:"thumbnail_url"`
	NoticeID       int    `json:"notice_id"`
	Date           string `json:"date"`
	DateEventStart string `json:"date_event_start"`
	DateEventEnd   string `json:"date_event_end"`
}

// EventNoticeResponse : /maplestory/v1/notice-event 응답
type EventNoticeResponse struct {
	EventNotice []EventNotice `json:"event_notice"`
}

// EventDetailResponse : /maplestory/v1/notice-event/detail 응답
type EventDetailResponse struct {
	Title          string `json:"title"`
	URL            string `json:"url"`
	Contents       string `json:"contents"`
	Date           string `json:"date"`
	DateEventStart string `json:"date_event_start"`
	DateEventEnd   string `json:"date_event_end"`
}

// GetEventNoticeList : 진행 중인 이벤트 목록을 조회합니다.
func (c *Client) GetEventNoticeList() (*EventNoticeResponse, error) {
	body, err := c.get("/maplestory/v1/notice-event", nil)
	if err != nil {
		return nil, err
	}

	var resp EventNoticeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("이벤트 목록 응답 파싱 실패: %w", err)
	}

	return &resp, nil
}

// GetEventDetail : 특정 이벤트의 상세 내용을 조회합니다.
func (c *Client) GetEventDetail(noticeID int) (*EventDetailResponse, error) {
	query := url.Values{}
	query.Set("notice_id", strconv.Itoa(noticeID))

	body, err := c.get("/maplestory/v1/notice-event/detail", query)
	if err != nil {
		return nil, err
	}

	var resp EventDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("이벤트 상세 응답 파싱 실패: %w", err)
	}

	return &resp, nil
}
