package brainprint

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	service := NewService(nil)
	assert.NotNil(t, service)
	assert.NotNil(t, service.profiles)
}

func TestTrackSession(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	// Track first session
	err := service.TrackSession(ctx, "user123", "visualization", true)
	require.NoError(t, err)

	// Track more sessions
	err = service.TrackSession(ctx, "user123", "visualization", true)
	require.NoError(t, err)

	err = service.TrackSession(ctx, "user123", "standard", true)
	require.NoError(t, err)

	// Get profile
	profile, err := service.GetBrainPrint(ctx, "user123")
	require.NoError(t, err)

	// Verify statistics
	assert.Equal(t, "user123", profile.UserID)
	assert.Equal(t, 3, profile.TotalSessions)
	assert.Equal(t, 2, profile.ByType["Visualization"])
	assert.Equal(t, 1, profile.ByType["Standard"])
}

func TestCalculateRecommendedType(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	// Track multiple sessions with different types
	service.TrackSession(ctx, "user1", "visualization", true)
	service.TrackSession(ctx, "user1", "visualization", true)
	service.TrackSession(ctx, "user1", "visualization", true)
	service.TrackSession(ctx, "user1", "standard", true)
	service.TrackSession(ctx, "user1", "standard", true)

	profile, err := service.GetBrainPrint(ctx, "user1")
	require.NoError(t, err)

	// Visualization should be recommended (most used)
	assert.Equal(t, "Visualization", profile.RecommendedType)
	assert.Equal(t, 3, profile.ByType["Visualization"])
	assert.Equal(t, 2, profile.ByType["Standard"])
}

func TestNormalizeExplanationType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"standard", "Standard"},
		{"Standard", "Standard"},
		{"visualization", "Visualization"},
		{"Visualization", "Visualization"},
		{"simple", "Simple"},
		{"Simple", "Simple"},
		{"analogy", "Analogy"},
		{"Analogy", "Analogy"},
		{"unknown", "Standard"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeExplanationType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewUserLearningProfile(t *testing.T) {
	profile := NewUserLearningProfile("user123")
	assert.Equal(t, "user123", profile.UserID)
	assert.Equal(t, 0, profile.TotalSessions)
	assert.NotNil(t, profile.ByType)
	assert.Equal(t, "Standard", profile.RecommendedType)
	assert.WithinDuration(t, time.Now(), profile.LastUpdated, time.Second)
}

func TestGetBrainPrintEmptyProfile(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	// Get profile for new user
	profile, err := service.GetBrainPrint(ctx, "newuser")
	require.NoError(t, err)

	assert.Equal(t, "newuser", profile.UserID)
	assert.Equal(t, 0, profile.TotalSessions)
	assert.Equal(t, "Standard", profile.RecommendedType)
}

