package discord

import (
	"fmt"
	"log"
	"maple-discord-bot/internal/format"
	"maple-discord-bot/internal/nexon"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	embedColorSuccess = 0x00A99D // 메이플 로고 계열 청록색
	embedColorError   = 0xE74C3C // 에러용 빨간색
)

var imgRegex = regexp.MustCompile(`(?i)<img\s+[^>]*src=["']?([^"'\s>]+)["']?`)

func extractImageURL(html string) string {
	matches := imgRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// StartSundayNoticeLoop : 백그라운드에서 매주 금요일 오전 10시 5분에 썬데이 메이플 공지를 체크하여 지정된 채널로 알림을 보냅니다.
func (b *Bot) StartSundayNoticeLoop() {
	if b.sundayChannelID == "" {
		log.Println("[썬데이알림] SUNDAY_CHANNEL_ID가 설정되지 않아 썬데이 메이플 백그라운드 알림을 시작하지 않습니다.")
		return
	}

	// 한국 표준시(KST) 타임존 설정
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Printf("[썬데이알림] Asia/Seoul 타임존 로드 실패: %v. UTC+9 고정 타임존을 사용합니다.", err)
		loc = time.FixedZone("KST", 9*60*60)
	}

	log.Printf("[썬데이알림] 썬데이 메이플 알림 루프 시작 (대상 채널: %s, 매주 금요일 오전 10시 5분 실행 예정)", b.sundayChannelID)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastRunYear, lastRunWeek int

	for range ticker.C {
		now := time.Now().In(loc)

		// 금요일 오전 10시 5분인지 검사 (10:05:00 ~ 10:05:59 사이)
		if now.Weekday() == time.Friday && now.Hour() == 10 && now.Minute() == 5 {
			year, week := now.ISOWeek()
			// 이번 주 금요일에 이미 실행했는지 체크 (중복 실행 방지)
			if lastRunYear == year && lastRunWeek == week {
				continue
			}

			log.Println("[썬데이알림] 금요일 오전 10시 5분 썬데이 메이플 공지 조회 시작...")
			lastRunYear = year
			lastRunWeek = week

			// 조회 및 전송 로직 비동기 실행
			go b.checkAndSendSundayNotice()
		}
	}
}

func (b *Bot) checkAndSendSundayNotice() {
	// 1. 이벤트 공지 목록 조회
	list, err := b.nexonClient.GetEventNoticeList()
	if err != nil {
		log.Printf("[썬데이알림] 이벤트 목록 조회 실패: %v", err)
		return
	}

	// 2. "스페셜 썬데이 메이플" 혹은 "썬데이 메이플" 공지 찾기
	var targetNotice *nexon.EventNotice
	for _, notice := range list.EventNotice {
		if notice.Title == "스페셜 썬데이 메이플" {
			targetNotice = &notice
			break
		}
	}
	if targetNotice == nil {
		for _, notice := range list.EventNotice {
			if strings.Contains(notice.Title, "썬데이 메이플") {
				targetNotice = &notice
				break
			}
		}
	}

	// 공지를 찾지 못한 경우
	if targetNotice == nil {
		log.Println("[썬데이알림] 이번 주 썬데이 메이플 공지를 찾을 수 없습니다. (아직 등록되지 않았거나 없음)")
		return
	}

	// 3. 공지 상세 조회
	detail, err := b.nexonClient.GetEventDetail(targetNotice.NoticeID)
	if err != nil {
		log.Printf("[썬데이알림] 이벤트 상세 조회 실패 (ID: %d): %v", targetNotice.NoticeID, err)
		return
	}

	// 4. HTML contents에서 이미지 URL 추출
	imageURL := extractImageURL(detail.Contents)

	// 5. 디스코드 임베드 빌드
	embed := &discordgo.MessageEmbed{
		Title: detail.Title,
		URL:   detail.URL,
		Color: embedColorSuccess,
		Description: fmt.Sprintf(
			"📅 **이벤트 기간**\n%s ~ %s\n\n🔗 [넥슨 공지사항 바로가기](%s)",
			format.DateTime(detail.DateEventStart),
			format.DateTime(detail.DateEventEnd),
			detail.URL,
		),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by NEXON Open API",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if imageURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: imageURL,
		}
	}

	// 6. 채널로 전송
	_, err = b.session.ChannelMessageSendEmbed(b.sundayChannelID, embed)
	if err != nil {
		log.Printf("[썬데이알림] 썬데이 메이플 알림 전송 실패: %v", err)
	} else {
		log.Printf("[썬데이알림] 썬데이 메이플 알림 전송 완료! (제목: %s)", detail.Title)
	}
}
