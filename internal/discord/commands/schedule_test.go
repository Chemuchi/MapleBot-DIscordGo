package commands

import (
	"strings"
	"testing"

	"maple-discord-bot/internal/nexon"
)

func stringPointer(value string) *string {
	return &value
}

func TestBuildContentFieldSpecialFormatting(t *testing.T) {
	contents := []nexon.SchedulerContent{
		{ContentName: "[일일 퀘스트] 문브릿지 조사", RegistrationFlag: "true", MaxCount: 100, QuestState: stringPointer("1")},
		{ContentName: "[일일 퀘스트] 세르니움 조사", RegistrationFlag: "true", QuestState: stringPointer("2")},
		{ContentName: "[일일 퀘스트] 도원경 오염 정화", RegistrationFlag: "true"},
		{ContentName: "[몬스터파크] 익스트림 몬스터파커에 도전해보겠나?", RegistrationFlag: "true", QuestState: stringPointer("2")},
		{ContentName: "에픽 던전 : 악몽선경", RegistrationFlag: "true", NowCount: 5},
		{ContentName: "에픽 던전 : 하이마운틴", RegistrationFlag: "true", NowCount: 0},
		{ContentName: "에픽 던전 : 앵글러 컴퍼니", RegistrationFlag: "false", NowCount: 3},
		{ContentName: "[길드] 지하 수로", RegistrationFlag: "true", NowCount: 13141},
		{ContentName: "[길드] 플래그 레이스", RegistrationFlag: "true", NowCount: 1000},
		{ContentName: "[일일 퀘스트] 숨겨진 퀘스트", RegistrationFlag: "false", QuestState: stringPointer("2")},
	}

	got := buildContentField(contents)
	wants := []string{
		"[일일 퀘스트] 문브릿지 조사 ❌",
		"[일일 퀘스트] 세르니움 조사 ✅",
		"[일일 퀘스트] 도원경 오염 정화 ❌",
		"[몬스터파크] 익스트림 몬스터파커에 도전해보겠나? ✅",
		"에픽 던전 : 악몽선경 ✅",
		"에픽 던전 : 하이마운틴 ❌",
		"[길드] 지하수로: 13141점",
		"[길드] 플래그레이스: 1000점",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("buildContentField() = %q, want to contain %q", got, want)
		}
	}
	if strings.Contains(got, "숨겨진 퀘스트") {
		t.Errorf("buildContentField() included an unregistered item: %q", got)
	}
	if strings.Contains(got, "앵글러 컴퍼니") {
		t.Errorf("buildContentField() included an unregistered epic dungeon: %q", got)
	}
}

func TestBuildBossFieldPlacesStatusAfterBoss(t *testing.T) {
	bosses := []nexon.SchedulerBossContent{
		{ContentName: "루시드", Difficulty: "hard", RegistrationFlag: "true", CompleteFlag: "true"},
		{ContentName: "스우", Difficulty: "normal", RegistrationFlag: "true", CompleteFlag: "false"},
		{ContentName: "데미안", Difficulty: "hard", RegistrationFlag: "false", CompleteFlag: "true"},
	}

	got := buildBossField(bosses)
	for _, want := range []string{"루시드 (하드) ✅", "스우 (노멀) ❌"} {
		if !strings.Contains(got, want) {
			t.Errorf("buildBossField() = %q, want to contain %q", got, want)
		}
	}
	if strings.Contains(got, "데미안") {
		t.Errorf("buildBossField() included an unregistered boss: %q", got)
	}
}
