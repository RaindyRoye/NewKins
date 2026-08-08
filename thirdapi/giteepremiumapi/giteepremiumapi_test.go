package giteepremiumapi

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{
			name:    "valid URL",
			uri:     "https://gitee.com/api/v5",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			uri:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("New() returned nil client")
			}
		})
	}
}

func TestNewDefault(t *testing.T) {
	client := NewDefault()
	if client == nil {
		t.Fatal("NewDefault() returned nil")
	}
}
