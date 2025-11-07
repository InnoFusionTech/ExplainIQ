package brainprint

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// StorageInterface defines the interface for storage operations needed by BrainPrint
type StorageInterface interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
}

// Service manages user learning profiles and BrainPrint recommendations
type Service struct {
	storage StorageInterface
	logger  *logrus.Logger
	mu      sync.RWMutex
	profiles map[string]*UserLearningProfile // In-memory cache
}

// NewService creates a new BrainPrint service
func NewService(storageClient StorageInterface) *Service {
	return &Service{
		storage:  storageClient,
		logger:   logrus.New(),
		profiles: make(map[string]*UserLearningProfile),
	}
}

// TrackSession tracks a completed session for a user
func (s *Service) TrackSession(ctx context.Context, userID string, explanationType string, success bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Normalize explanation type
	normalizedType := NormalizeExplanationType(explanationType)

	// Get or create profile
	profile, err := s.getProfile(ctx, userID)
	if err != nil {
		profile = NewUserLearningProfile(userID)
	}

	// Update statistics
	profile.TotalSessions++
	profile.ByType[normalizedType]++
	profile.LastUpdated = time.Now()

	// Update success rate (simple calculation: track successful sessions per type)
	if success {
		// Initialize success count if not exists
		if profile.SuccessRate == nil {
			profile.SuccessRate = make(map[string]float64)
		}
		// Simple success rate: increment by 1 for each successful session
		// In a real implementation, you'd track success/failure counts separately
		currentSuccess := profile.SuccessRate[normalizedType]
		profile.SuccessRate[normalizedType] = currentSuccess + 1.0
	}

	// Calculate recommended type
	profile.RecommendedType = s.calculateRecommendedType(profile)

	// Save profile
	if err := s.saveProfile(ctx, profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"userID":          userID,
		"explanationType": normalizedType,
		"totalSessions":   profile.TotalSessions,
		"recommendedType": profile.RecommendedType,
	}).Info("Session tracked for BrainPrint")

	return nil
}

// GetBrainPrint retrieves the BrainPrint for a user
func (s *Service) GetBrainPrint(ctx context.Context, userID string) (*UserLearningProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, err := s.getProfile(ctx, userID)
	if err != nil {
		// Return empty profile if not found
		return NewUserLearningProfile(userID), nil
	}

	return profile, nil
}

// getProfile retrieves a profile from storage or cache
func (s *Service) getProfile(ctx context.Context, userID string) (*UserLearningProfile, error) {
	// Check cache first
	if profile, exists := s.profiles[userID]; exists {
		return profile, nil
	}

	// Try to load from storage
	if s.storage != nil {
		key := fmt.Sprintf("brainprint:%s", userID)
		data, err := s.storage.Get(ctx, key)
		if err == nil && len(data) > 0 {
			var profile UserLearningProfile
			if err := json.Unmarshal(data, &profile); err == nil {
				s.profiles[userID] = &profile
				return &profile, nil
			}
		}
	}

	// Create new profile if not found
	profile := NewUserLearningProfile(userID)
	s.profiles[userID] = profile
	return profile, nil
}

// saveProfile saves a profile to storage
func (s *Service) saveProfile(ctx context.Context, profile *UserLearningProfile) error {
	// Update cache
	s.profiles[profile.UserID] = profile

	// Save to storage if available
	if s.storage != nil {
		key := fmt.Sprintf("brainprint:%s", profile.UserID)
		data, err := json.Marshal(profile)
		if err != nil {
			return fmt.Errorf("failed to marshal profile: %w", err)
		}

		if err := s.storage.Set(ctx, key, data); err != nil {
			return fmt.Errorf("failed to save profile to storage: %w", err)
		}
	}

	return nil
}

// calculateRecommendedType determines the recommended explanation type based on usage and success rate
func (s *Service) calculateRecommendedType(profile *UserLearningProfile) string {
	if profile.TotalSessions == 0 {
		return string(ExplanationTypeStandard)
	}

	// Calculate weighted score: usage count + success rate
	type Score struct {
		Type  string
		Score float64
	}

	scores := make([]Score, 0, len(profile.ByType))

	for explanationType, count := range profile.ByType {
		// Base score from usage count
		score := float64(count)

		// Add success rate bonus (if available)
		if profile.SuccessRate != nil {
			if successRate, exists := profile.SuccessRate[explanationType]; exists {
				// Weight success rate: higher success rate = higher score
				score += successRate * 0.5
			}
		}

		scores = append(scores, Score{
			Type:  explanationType,
			Score: score,
		})
	}

	// Find the type with highest score
	if len(scores) == 0 {
		return string(ExplanationTypeStandard)
	}

	maxScore := scores[0]
	for _, score := range scores[1:] {
		if score.Score > maxScore.Score {
			maxScore = score
		}
	}

	// If user has tried multiple types, suggest the least used one to encourage exploration
	// But only if they have enough sessions (at least 5)
	if profile.TotalSessions >= 5 {
		// Find least used type
		minCount := profile.TotalSessions
		leastUsedType := ""
		for explanationType, count := range profile.ByType {
			if count < minCount {
				minCount = count
				leastUsedType = explanationType
			}
		}

		// If least used type exists and is significantly less used, suggest it
		if leastUsedType != "" && float64(minCount) < maxScore.Score*0.5 {
			return leastUsedType
		}
	}

	return maxScore.Type
}

// GetStats returns statistics about BrainPrint usage
func (s *Service) GetStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_profiles": len(s.profiles),
		"total_sessions": 0,
		"by_type":        make(map[string]int),
	}

	for _, profile := range s.profiles {
		stats["total_sessions"] = stats["total_sessions"].(int) + profile.TotalSessions
		for explanationType, count := range profile.ByType {
			byType := stats["by_type"].(map[string]int)
			byType[explanationType] += count
		}
	}

	return stats, nil
}

