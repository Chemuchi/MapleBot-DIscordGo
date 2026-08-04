package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestModalTextValue(t *testing.T) {
	data := discordgo.ModalSubmitInteractionData{
		Components: []discordgo.MessageComponent{
			&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				&discordgo.TextInput{CustomID: apiKeyInputID, Value: "test-api-key"},
			}},
		},
	}

	if got := modalTextValue(data, apiKeyInputID); got != "test-api-key" {
		t.Fatalf("modalTextValue() = %q, want %q", got, "test-api-key")
	}
	if got := modalTextValue(data, "unknown"); got != "" {
		t.Fatalf("unknown modalTextValue() = %q, want empty", got)
	}
}

func TestInteractionUserID(t *testing.T) {
	tests := []struct {
		name        string
		interaction *discordgo.InteractionCreate
		want        string
	}{
		{
			name: "guild member",
			interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Member: &discordgo.Member{User: &discordgo.User{ID: "guild-user"}},
			}},
			want: "guild-user",
		},
		{
			name: "direct message user",
			interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				User: &discordgo.User{ID: "dm-user"},
			}},
			want: "dm-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interactionUserID(tt.interaction); got != tt.want {
				t.Fatalf("interactionUserID() = %q, want %q", got, tt.want)
			}
		})
	}
}
