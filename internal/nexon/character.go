package nexon

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// CharacterBasic : /maplestory/v1/character/basic 응답
type CharacterBasic struct {
	Date                 *string `json:"date"`
	CharacterName        string  `json:"character_name"`
	WorldName            string  `json:"world_name"`
	CharacterGender      string  `json:"character_gender"`
	CharacterClass       string  `json:"character_class"`
	CharacterClassLevel  string  `json:"character_class_level"`
	CharacterLevel       int     `json:"character_level"`
	CharacterExp         int64   `json:"character_exp"`
	CharacterExpRate     string  `json:"character_exp_rate"`
	CharacterGuildName   string  `json:"character_guild_name"`
	CharacterImage       string  `json:"character_image"`
	CharacterDateCreate  string  `json:"character_date_create"`
	AccessFlag           string  `json:"access_flag"`
	LiberationQuestClear string  `json:"liberation_quest_clear"`
}

// ocidResponse : /maplestory/v1/id 응답
type ocidResponse struct {
	Ocid string `json:"ocid"`
}

// GetOcid : 캐릭터 닉네임으로 OCID를 조회합니다 (1차 API 호출).
func (c *Client) GetOcid(characterName string) (string, error) {
	query := url.Values{}
	query.Set("character_name", characterName)

	body, err := c.get("/maplestory/v1/id", query)
	if err != nil {
		return "", err
	}

	var resp ocidResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("OCID 응답 파싱 실패: %w", err)
	}

	return resp.Ocid, nil
}

// GetCharacterBasic : OCID로 캐릭터 기본 정보를 조회합니다 (2차 API 호출).
func (c *Client) GetCharacterBasic(ocid string) (*CharacterBasic, error) {
	query := url.Values{}
	query.Set("ocid", ocid)

	body, err := c.get("/maplestory/v1/character/basic", query)
	if err != nil {
		return nil, err
	}

	var basic CharacterBasic
	if err := json.Unmarshal(body, &basic); err != nil {
		return nil, fmt.Errorf("캐릭터 정보 응답 파싱 실패: %w", err)
	}

	return &basic, nil
}

// SearchCharacterByName : 닉네임 하나로 OCID 조회 -> 기본 정보 조회까지 한번에 처리하는
// 편의 함수. discord/commands 쪽에서는 이 함수 하나만 호출하면 됩니다.
func (c *Client) SearchCharacterByName(characterName string) (*CharacterBasic, error) {
	ocid, err := c.GetOcid(characterName)
	if err != nil {
		return nil, err
	}
	return c.GetCharacterBasic(ocid)
}
