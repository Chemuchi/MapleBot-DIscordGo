package nexon

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// SchedulerContent : daily_contents / weekly_contents 배열의 원소.
// type이 "contents"인 경우 quest_state는 보통 null이고, "quest"인 경우
// quest_state에 값이 들어옵니다 (정확한 값의 의미는 registration 상태에 따라
// 다를 수 있어 IsRegistered()만 신뢰도 높게 판단하고 있습니다).
type SchedulerContent struct {
	ContentName      string  `json:"content_name"`
	Type             string  `json:"type"`              // "contents" | "quest"
	RegistrationFlag string  `json:"registration_flag"` // "true" | "false" (문자열)
	NowCount         int     `json:"now_count"`
	MaxCount         int     `json:"max_count"`
	QuestState       *string `json:"quest_state"` // nullable
}

// IsRegistered : 캐릭터의 메이플 스케줄러에 등록되어 있는 콘텐츠/퀘스트인지 여부.
func (c SchedulerContent) IsRegistered() bool {
	return c.RegistrationFlag == "true"
}

// HasProgress : max_count가 0보다 커서 now_count/max_count로 진행률을 표현할 수 있는지 여부.
func (c SchedulerContent) HasProgress() bool {
	return c.MaxCount > 0
}

// SchedulerBossContent : boss_contents 배열의 원소.
// complete_flag는 "이번 주기(일간/주간/월간) 내 처치 완료 여부"를 명확하게 나타냅니다.
type SchedulerBossContent struct {
	ContentName      string `json:"content_name"`
	Difficulty       string `json:"difficulty"` // easy | normal | hard | chaos | extreme
	Cycle            string `json:"cycle"`      // bossDaily | bossWeekly | bossMonthly
	ListOrderNo      int    `json:"list_order_no"`
	RegistrationFlag string `json:"registration_flag"`
	CompleteFlag     string `json:"complete_flag"`
}

// IsRegistered : 보스 격파 리스트에 등록되어 있는지 여부.
func (b SchedulerBossContent) IsRegistered() bool {
	return b.RegistrationFlag == "true"
}

// IsComplete : 해당 주기 내에 처치를 완료했는지 여부.
func (b SchedulerBossContent) IsComplete() bool {
	return b.CompleteFlag == "true"
}

// SchedulerCharacterState : /maplestory/v1/scheduler/character-state 응답
type SchedulerCharacterState struct {
	Date                      string                 `json:"date"`
	CharacterName             string                 `json:"character_name"`
	WorldName                 string                 `json:"world_name"`
	CharacterLevel            int                    `json:"character_level"`
	CharacterClass            string                 `json:"character_class"`
	DailyContents             []SchedulerContent     `json:"daily_contents"`
	WeeklyContents            []SchedulerContent     `json:"weekly_contents"`
	BossContents              []SchedulerBossContent `json:"boss_contents"`
	WeeklyBossClearCount      int                    `json:"weekly_boss_clear_count"`
	WeeklyBossClearLimitCount int                    `json:"weekly_boss_clear_limit_count"`
}

// GetSchedulerCharacterState : OCID로 캐릭터의 메이플 스케줄러 달성 현황을 조회합니다.
// date가 빈 문자열이 아니면 해당 날짜(YYYY-MM-DD) 기준 과거 정보를 조회합니다
// (넥슨 공지 기준 조회 요청일로부터 최대 14일 전까지 가능).
func (c *Client) GetSchedulerCharacterState(ocid string, date string) (*SchedulerCharacterState, error) {
	query := url.Values{}
	query.Set("ocid", ocid)
	if date != "" {
		query.Set("date", date)
	}

	body, err := c.get("/maplestory/v1/scheduler/character-state", query)
	if err != nil {
		return nil, err
	}

	var state SchedulerCharacterState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("스케줄러 응답 파싱 실패: %w", err)
	}

	return &state, nil
}

// SearchSchedulerByName : 닉네임 하나로 OCID 조회 -> 스케줄러 현황 조회까지 한번에 처리.
func (c *Client) SearchSchedulerByName(characterName string, date string) (*SchedulerCharacterState, error) {
	ocid, err := c.GetOcid(characterName)
	if err != nil {
		return nil, err
	}
	return c.GetSchedulerCharacterState(ocid, date)
}
