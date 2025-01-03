package shorttermmemory

type Usage struct {
	// Number of tokens in the generated completion.
	CompletionTokens int64 `json:"completion_tokens"`
	// Number of tokens in the prompt.
	PromptTokens int64 `json:"prompt_tokens"`
	// Total number of tokens used in the request (prompt + completion).
	TotalTokens int64 `json:"total_tokens"`
	// Breakdown of tokens used in a completion.
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
	// Breakdown of tokens used in the prompt.
	PromptTokensDetails PromptTokensDetails `json:"prompt_tokens_details"`
}

func (u *Usage) AddUsage(usage *Usage) {
	u.CompletionTokens += usage.CompletionTokens
	u.PromptTokens += usage.PromptTokens
	u.TotalTokens += usage.TotalTokens
	u.CompletionTokensDetails.AddUsage(&usage.CompletionTokensDetails)
	u.PromptTokensDetails.AddUsage(&usage.PromptTokensDetails)
}

type CompletionTokensDetails struct {
	// When using Predicted Outputs, the number of tokens in the prediction that
	// appeared in the completion.
	AcceptedPredictionTokens int64 `json:"accepted_prediction_tokens"`
	// Audio input tokens generated by the model.
	AudioTokens int64 `json:"audio_tokens"`
	// Tokens generated by the model for reasoning.
	ReasoningTokens int64 `json:"reasoning_tokens"`
	// When using Predicted Outputs, the number of tokens in the prediction that did
	// not appear in the completion. However, like reasoning tokens, these tokens are
	// still counted in the total completion tokens for purposes of billing, output,
	// and context window limits.
	RejectedPredictionTokens int64 `json:"rejected_prediction_tokens"`
}

func (c *CompletionTokensDetails) AddUsage(details *CompletionTokensDetails) {
	c.AcceptedPredictionTokens += details.AcceptedPredictionTokens
	c.AudioTokens += details.AudioTokens
	c.ReasoningTokens += details.ReasoningTokens
	c.RejectedPredictionTokens += details.RejectedPredictionTokens
}

type PromptTokensDetails struct {
	// Audio input tokens present in the prompt.
	AudioTokens int64 `json:"audio_tokens"`
	// Cached tokens present in the prompt.
	CachedTokens int64 `json:"cached_tokens"`
}

func (p *PromptTokensDetails) AddUsage(details *PromptTokensDetails) {
	p.AudioTokens += details.AudioTokens
	p.CachedTokens += details.CachedTokens
}
