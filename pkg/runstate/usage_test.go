package runstate

import (
	"testing"
)

func TestUsage_AddUsage(t *testing.T) {
	tests := []struct {
		name     string
		initial  Usage
		add      Usage
		expected Usage
	}{
		{
			name: "basic addition",
			initial: Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 5,
					AudioTokens:              2,
					ReasoningTokens:          3,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  1,
					CachedTokens: 19,
				},
			},
			add: Usage{
				CompletionTokens: 15,
				PromptTokens:     25,
				TotalTokens:      40,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 7,
					AudioTokens:              3,
					ReasoningTokens:          5,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  2,
					CachedTokens: 23,
				},
			},
			expected: Usage{
				CompletionTokens: 25,
				PromptTokens:     45,
				TotalTokens:      70,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 12,
					AudioTokens:              5,
					ReasoningTokens:          8,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  3,
					CachedTokens: 42,
				},
			},
		},
		{
			name:    "zero values",
			initial: Usage{},
			add: Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 5,
					AudioTokens:              2,
					ReasoningTokens:          3,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  1,
					CachedTokens: 19,
				},
			},
			expected: Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 5,
					AudioTokens:              2,
					ReasoningTokens:          3,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  1,
					CachedTokens: 19,
				},
			},
		},
		{
			name: "adding zero values",
			initial: Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 5,
					AudioTokens:              2,
					ReasoningTokens:          3,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  1,
					CachedTokens: 19,
				},
			},
			add: Usage{},
			expected: Usage{
				CompletionTokens: 10,
				PromptTokens:     20,
				TotalTokens:      30,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 5,
					AudioTokens:              2,
					ReasoningTokens:          3,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  1,
					CachedTokens: 19,
				},
			},
		},
		{
			name: "large numbers",
			initial: Usage{
				CompletionTokens: 1000000,
				PromptTokens:     2000000,
				TotalTokens:      3000000,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 500000,
					AudioTokens:              200000,
					ReasoningTokens:          300000,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  100000,
					CachedTokens: 1900000,
				},
			},
			add: Usage{
				CompletionTokens: 1500000,
				PromptTokens:     2500000,
				TotalTokens:      4000000,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 700000,
					AudioTokens:              300000,
					ReasoningTokens:          500000,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  200000,
					CachedTokens: 2300000,
				},
			},
			expected: Usage{
				CompletionTokens: 2500000,
				PromptTokens:     4500000,
				TotalTokens:      7000000,
				CompletionTokensDetails: CompletionTokensDetails{
					AcceptedPredictionTokens: 1200000,
					AudioTokens:              500000,
					ReasoningTokens:          800000,
					RejectedPredictionTokens: 0,
				},
				PromptTokensDetails: PromptTokensDetails{
					AudioTokens:  300000,
					CachedTokens: 4200000,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.AddUsage(&tt.add)

			if tt.initial.CompletionTokens != tt.expected.CompletionTokens {
				t.Errorf("CompletionTokens = %v, want %v", tt.initial.CompletionTokens, tt.expected.CompletionTokens)
			}
			if tt.initial.PromptTokens != tt.expected.PromptTokens {
				t.Errorf("PromptTokens = %v, want %v", tt.initial.PromptTokens, tt.expected.PromptTokens)
			}
			if tt.initial.TotalTokens != tt.expected.TotalTokens {
				t.Errorf("TotalTokens = %v, want %v", tt.initial.TotalTokens, tt.expected.TotalTokens)
			}

			// Check CompletionTokensDetails
			if tt.initial.CompletionTokensDetails.AcceptedPredictionTokens != tt.expected.CompletionTokensDetails.AcceptedPredictionTokens {
				t.Errorf("AcceptedPredictionTokens = %v, want %v",
					tt.initial.CompletionTokensDetails.AcceptedPredictionTokens,
					tt.expected.CompletionTokensDetails.AcceptedPredictionTokens)
			}
			if tt.initial.CompletionTokensDetails.AudioTokens != tt.expected.CompletionTokensDetails.AudioTokens {
				t.Errorf("CompletionTokensDetails.AudioTokens = %v, want %v",
					tt.initial.CompletionTokensDetails.AudioTokens,
					tt.expected.CompletionTokensDetails.AudioTokens)
			}
			if tt.initial.CompletionTokensDetails.ReasoningTokens != tt.expected.CompletionTokensDetails.ReasoningTokens {
				t.Errorf("ReasoningTokens = %v, want %v",
					tt.initial.CompletionTokensDetails.ReasoningTokens,
					tt.expected.CompletionTokensDetails.ReasoningTokens)
			}
			if tt.initial.CompletionTokensDetails.RejectedPredictionTokens != tt.expected.CompletionTokensDetails.RejectedPredictionTokens {
				t.Errorf("RejectedPredictionTokens = %v, want %v",
					tt.initial.CompletionTokensDetails.RejectedPredictionTokens,
					tt.expected.CompletionTokensDetails.RejectedPredictionTokens)
			}

			// Check PromptTokensDetails
			if tt.initial.PromptTokensDetails.AudioTokens != tt.expected.PromptTokensDetails.AudioTokens {
				t.Errorf("PromptTokensDetails.AudioTokens = %v, want %v",
					tt.initial.PromptTokensDetails.AudioTokens,
					tt.expected.PromptTokensDetails.AudioTokens)
			}
			if tt.initial.PromptTokensDetails.CachedTokens != tt.expected.PromptTokensDetails.CachedTokens {
				t.Errorf("CachedTokens = %v, want %v",
					tt.initial.PromptTokensDetails.CachedTokens,
					tt.expected.PromptTokensDetails.CachedTokens)
			}
		})
	}
}

func TestCompletionTokensDetails_AddUsage(t *testing.T) {
	tests := []struct {
		name     string
		initial  CompletionTokensDetails
		add      CompletionTokensDetails
		expected CompletionTokensDetails
	}{
		{
			name: "basic addition",
			initial: CompletionTokensDetails{
				AcceptedPredictionTokens: 10,
				AudioTokens:              5,
				ReasoningTokens:          15,
				RejectedPredictionTokens: 2,
			},
			add: CompletionTokensDetails{
				AcceptedPredictionTokens: 5,
				AudioTokens:              3,
				ReasoningTokens:          7,
				RejectedPredictionTokens: 1,
			},
			expected: CompletionTokensDetails{
				AcceptedPredictionTokens: 15,
				AudioTokens:              8,
				ReasoningTokens:          22,
				RejectedPredictionTokens: 3,
			},
		},
		{
			name:    "zero values",
			initial: CompletionTokensDetails{},
			add: CompletionTokensDetails{
				AcceptedPredictionTokens: 5,
				AudioTokens:              3,
				ReasoningTokens:          7,
				RejectedPredictionTokens: 1,
			},
			expected: CompletionTokensDetails{
				AcceptedPredictionTokens: 5,
				AudioTokens:              3,
				ReasoningTokens:          7,
				RejectedPredictionTokens: 1,
			},
		},
		{
			name: "adding zero values",
			initial: CompletionTokensDetails{
				AcceptedPredictionTokens: 10,
				AudioTokens:              5,
				ReasoningTokens:          15,
				RejectedPredictionTokens: 2,
			},
			add: CompletionTokensDetails{},
			expected: CompletionTokensDetails{
				AcceptedPredictionTokens: 10,
				AudioTokens:              5,
				ReasoningTokens:          15,
				RejectedPredictionTokens: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.AddUsage(&tt.add)

			if tt.initial.AcceptedPredictionTokens != tt.expected.AcceptedPredictionTokens {
				t.Errorf("AcceptedPredictionTokens = %v, want %v",
					tt.initial.AcceptedPredictionTokens,
					tt.expected.AcceptedPredictionTokens)
			}
			if tt.initial.AudioTokens != tt.expected.AudioTokens {
				t.Errorf("AudioTokens = %v, want %v",
					tt.initial.AudioTokens,
					tt.expected.AudioTokens)
			}
			if tt.initial.ReasoningTokens != tt.expected.ReasoningTokens {
				t.Errorf("ReasoningTokens = %v, want %v",
					tt.initial.ReasoningTokens,
					tt.expected.ReasoningTokens)
			}
			if tt.initial.RejectedPredictionTokens != tt.expected.RejectedPredictionTokens {
				t.Errorf("RejectedPredictionTokens = %v, want %v",
					tt.initial.RejectedPredictionTokens,
					tt.expected.RejectedPredictionTokens)
			}
		})
	}
}

func TestPromptTokensDetails_AddUsage(t *testing.T) {
	tests := []struct {
		name     string
		initial  PromptTokensDetails
		add      PromptTokensDetails
		expected PromptTokensDetails
	}{
		{
			name: "basic addition",
			initial: PromptTokensDetails{
				AudioTokens:  10,
				CachedTokens: 20,
			},
			add: PromptTokensDetails{
				AudioTokens:  5,
				CachedTokens: 10,
			},
			expected: PromptTokensDetails{
				AudioTokens:  15,
				CachedTokens: 30,
			},
		},
		{
			name:    "zero values",
			initial: PromptTokensDetails{},
			add: PromptTokensDetails{
				AudioTokens:  5,
				CachedTokens: 10,
			},
			expected: PromptTokensDetails{
				AudioTokens:  5,
				CachedTokens: 10,
			},
		},
		{
			name: "adding zero values",
			initial: PromptTokensDetails{
				AudioTokens:  10,
				CachedTokens: 20,
			},
			add: PromptTokensDetails{},
			expected: PromptTokensDetails{
				AudioTokens:  10,
				CachedTokens: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.AddUsage(&tt.add)

			if tt.initial.AudioTokens != tt.expected.AudioTokens {
				t.Errorf("AudioTokens = %v, want %v",
					tt.initial.AudioTokens,
					tt.expected.AudioTokens)
			}
			if tt.initial.CachedTokens != tt.expected.CachedTokens {
				t.Errorf("CachedTokens = %v, want %v",
					tt.initial.CachedTokens,
					tt.expected.CachedTokens)
			}
		})
	}
}
