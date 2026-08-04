package commands

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type fakeSchedulerAPIKeyFinder struct {
	wantUserID string
	apiKey     string
	err        error
	called     bool
}

func (f *fakeSchedulerAPIKeyFinder) Find(_ context.Context, userID string) (string, error) {
	f.called = true
	if userID != f.wantUserID {
		return "", errors.New("unexpected user ID")
	}
	return f.apiKey, f.err
}

func TestLookupSchedulerAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		finderErr error
		wantKey   string
		wantErr   error
	}{
		{
			name:    "registered user",
			apiKey:  "decrypted-user-api-key",
			wantKey: "decrypted-user-api-key",
		},
		{
			name:      "unregistered user",
			finderErr: sql.ErrNoRows,
			wantErr:   sql.ErrNoRows,
		},
		{
			name:      "database error",
			finderErr: errors.New("database unavailable"),
			wantErr:   errors.New("database unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := &fakeSchedulerAPIKeyFinder{
				wantUserID: "discord-user-1234",
				apiKey:     tt.apiKey,
				err:        tt.finderErr,
			}
			got, err := lookupSchedulerAPIKey(context.Background(), finder, "discord-user-1234")
			if !finder.called {
				t.Fatal("API Key 저장소가 호출되지 않았습니다")
			}
			if got != tt.wantKey {
				t.Fatalf("lookupSchedulerAPIKey() key = %q, want %q", got, tt.wantKey)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("lookupSchedulerAPIKey() error = %v", err)
			}
			if tt.wantErr != nil && (err == nil || err.Error() != tt.wantErr.Error()) {
				t.Fatalf("lookupSchedulerAPIKey() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
