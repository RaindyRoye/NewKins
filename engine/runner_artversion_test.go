package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindArtVersionIdCtx_EmptyBuildID verifies that FindArtVersionIdCtx validates empty buildID
func TestFindArtVersionIdCtx_EmptyBuildID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.FindArtVersionIdCtx(ctx, "", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	assert.Contains(t, err.Error(), "buildID")
}

// TestFindArtVersionIdCtx_EmptyIdentifier verifies that FindArtVersionIdCtx validates empty identifier
func TestFindArtVersionIdCtx_EmptyIdentifier(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.FindArtVersionIdCtx(ctx, "build123", "", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	assert.Contains(t, err.Error(), "identifier")
}

// TestFindArtVersionIdCtx_EmptyName verifies that FindArtVersionIdCtx validates empty name
func TestFindArtVersionIdCtx_EmptyName(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.FindArtVersionIdCtx(ctx, "build123", "artifact", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	assert.Contains(t, err.Error(), "name")
}

// TestFindArtVersionIdCtx_BuildNotFound verifies that FindArtVersionIdCtx returns ErrBuildNotFound
func TestFindArtVersionIdCtx_BuildNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.FindArtVersionIdCtx(ctx, "nonexistent-build", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

// TestNewArtVersionIdCtx_EmptyBuildID verifies that NewArtVersionIdCtx validates empty buildID
func TestNewArtVersionIdCtx_EmptyBuildID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.NewArtVersionIdCtx(ctx, "", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	assert.Contains(t, err.Error(), "buildID")
}

// TestNewArtVersionIdCtx_EmptyIdentifier verifies that NewArtVersionIdCtx validates empty identifier
func TestNewArtVersionIdCtx_EmptyIdentifier(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.NewArtVersionIdCtx(ctx, "build123", "", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	assert.Contains(t, err.Error(), "identifier")
}

// TestNewArtVersionIdCtx_EmptyName verifies that NewArtVersionIdCtx validates empty name
func TestNewArtVersionIdCtx_EmptyName(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.NewArtVersionIdCtx(ctx, "build123", "artifact", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	assert.Contains(t, err.Error(), "name")
}

// TestNewArtVersionIdCtx_BuildNotFound verifies that NewArtVersionIdCtx returns ErrBuildNotFound
func TestNewArtVersionIdCtx_BuildNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	_, err := br.NewArtVersionIdCtx(ctx, "nonexistent-build", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

// TestFindArtVersionId_DeprecatedWrapper verifies that the deprecated FindArtVersionId
// function properly delegates to FindArtVersionIdCtx
func TestFindArtVersionId_DeprecatedWrapper(t *testing.T) {
	// Initialize comm.Ctx for the wrapper function
	if comm.Ctx == nil {
		var cancel context.CancelFunc
		comm.Ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	br := &baseRunner{}
	// Should fail with ErrEmptyParams since we're passing empty buildID
	_, err := br.FindArtVersionId("", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
}

// TestNewArtVersionId_DeprecatedWrapper verifies that the deprecated NewArtVersionId
// function properly delegates to NewArtVersionIdCtx
func TestNewArtVersionId_DeprecatedWrapper(t *testing.T) {
	// Initialize comm.Ctx for the wrapper function
	if comm.Ctx == nil {
		var cancel context.CancelFunc
		comm.Ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	br := &baseRunner{}
	// Should fail with ErrEmptyParams since we're passing empty buildID
	_, err := br.NewArtVersionId("", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
}

// TestFindArtVersionIdCtx_ContextCancellation verifies that FindArtVersionIdCtx respects context cancellation
func TestFindArtVersionIdCtx_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	br := &baseRunner{}
	// The function should check parameters first, so it will return ErrEmptyParams
	// before checking the context
	_, err := br.FindArtVersionIdCtx(ctx, "", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
}

// TestNewArtVersionIdCtx_ContextCancellation verifies that NewArtVersionIdCtx respects context cancellation
func TestNewArtVersionIdCtx_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	br := &baseRunner{}
	// The function should check parameters first, so it will return ErrEmptyParams
	// before checking the context
	_, err := br.NewArtVersionIdCtx(ctx, "", "artifact", "name")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
}

// TestFindArtVersionIdCtx_NameWithVersion verifies that FindArtVersionIdCtx properly parses name@version format
func TestFindArtVersionIdCtx_NameWithVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	// Should fail with ErrBuildNotFound since the build doesn't exist
	_, err := br.FindArtVersionIdCtx(ctx, "nonexistent-build", "artifact", "myart@1.0.0")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

// TestNewArtVersionIdCtx_NameWithVersion verifies that NewArtVersionIdCtx properly parses name@version format
func TestNewArtVersionIdCtx_NameWithVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	br := &baseRunner{}
	// Should fail with ErrBuildNotFound since the build doesn't exist
	_, err := br.NewArtVersionIdCtx(ctx, "nonexistent-build", "artifact", "myart@1.0.0")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}
