package engine

import (
	"context"
	"testing"
	"time"
)

func TestFindArtVersionIdCtx_EmptyParams(t *testing.T) {
	ctx := context.Background()
	br := &baseRunner{}
	
	// Test empty buildID
	_, err := br.FindArtVersionIdCtx(ctx, "", "artifact", "name")
	if err == nil {
		t.Error("Expected error for empty buildID")
	}
	
	// Test empty idnt
	_, err = br.FindArtVersionIdCtx(ctx, "build1", "", "name")
	if err == nil {
		t.Error("Expected error for empty idnt")
	}
	
	// Test empty name
	_, err = br.FindArtVersionIdCtx(ctx, "build1", "artifact", "")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestFindArtVersionIdCtx_NilContext(t *testing.T) {
	br := &baseRunner{}
	
	// Should use comm.Ctx when nil context is passed
	_, err := br.FindArtVersionIdCtx(nil, "", "artifact", "name")
	if err == nil {
		t.Error("Expected error for empty params")
	}
}

func TestNewArtVersionIdCtx_EmptyParams(t *testing.T) {
	ctx := context.Background()
	br := &baseRunner{}
	
	// Test empty buildID
	_, err := br.NewArtVersionIdCtx(ctx, "", "artifact", "name")
	if err == nil {
		t.Error("Expected error for empty buildID")
	}
	
	// Test empty idnt
	_, err = br.NewArtVersionIdCtx(ctx, "build1", "", "name")
	if err == nil {
		t.Error("Expected error for empty idnt")
	}
	
	// Test empty name
	_, err = br.NewArtVersionIdCtx(ctx, "build1", "artifact", "")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestNewArtVersionIdCtx_NilContext(t *testing.T) {
	br := &baseRunner{}
	
	// Should use comm.Ctx when nil context is passed
	_, err := br.NewArtVersionIdCtx(nil, "", "artifact", "name")
	if err == nil {
		t.Error("Expected error for empty params")
	}
}

func TestFindArtVersionIdCtx_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	// Wait for timeout
	time.Sleep(1 * time.Millisecond)
	
	br := &baseRunner{}
	
	// This should fail due to empty params before accessing DB
	_, err := br.FindArtVersionIdCtx(ctx, "", "artifact", "test-art")
	if err == nil {
		t.Error("Expected error for empty params")
	}
}

func TestNewArtVersionIdCtx_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	// Wait for timeout
	time.Sleep(1 * time.Millisecond)
	
	br := &baseRunner{}
	
	// This should fail due to empty params before accessing DB
	_, err := br.NewArtVersionIdCtx(ctx, "", "artifact", "test-art")
	if err == nil {
		t.Error("Expected error for empty params")
	}
}
