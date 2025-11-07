package brainprint

import (
	"time"
)

// ExplanationType represents the type of explanation
type ExplanationType string

const (
	ExplanationTypeStandard      ExplanationType = "Standard"
	ExplanationTypeVisualization ExplanationType = "Visualization"
	ExplanationTypeSimple        ExplanationType = "Simple"
	ExplanationTypeAnalogy       ExplanationType = "Analogy"
)

// UserLearningProfile represents a user's learning profile and preferences
type UserLearningProfile struct {
	UserID          string                `json:"userID"`
	TotalSessions   int                   `json:"totalSessions"`
	ByType          map[string]int         `json:"byType"` // {"Standard": 4, "Visualization": 6}
	LastUpdated     time.Time             `json:"lastUpdated"`
	RecommendedType string                `json:"recommendedType"`
	SuccessRate     map[string]float64    `json:"successRate,omitempty"` // Success rate per type
	Engagement      map[string]float64    `json:"engagement,omitempty"` // Engagement metrics per type
}

// NewUserLearningProfile creates a new user learning profile
func NewUserLearningProfile(userID string) *UserLearningProfile {
	return &UserLearningProfile{
		UserID:          userID,
		TotalSessions:   0,
		ByType:          make(map[string]int),
		LastUpdated:     time.Now(),
		RecommendedType: string(ExplanationTypeStandard), // Default recommendation
		SuccessRate:     make(map[string]float64),
		Engagement:      make(map[string]float64),
	}
}

// NormalizeExplanationType normalizes explanation type strings to standard format
func NormalizeExplanationType(explanationType string) string {
	switch explanationType {
	case "standard", "Standard":
		return string(ExplanationTypeStandard)
	case "visualization", "Visualization":
		return string(ExplanationTypeVisualization)
	case "simple", "Simple":
		return string(ExplanationTypeSimple)
	case "analogy", "Analogy":
		return string(ExplanationTypeAnalogy)
	default:
		return string(ExplanationTypeStandard)
	}
}

