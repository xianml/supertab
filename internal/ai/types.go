package ai

import "time"

// Provider represents different AI service providers
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
	ProviderGroq      Provider = "groq"
)

// CompletionRequest represents a request for command completion
type CompletionRequest struct {
	Input   string  `json:"input"`
	Context Context `json:"context"`
}

// PredictionRequest represents a request for command prediction
type PredictionRequest struct {
	History []HistoryEntry `json:"history"`
	Context Context        `json:"context"`
}

// Context contains environmental information for AI requests
type Context struct {
	User       string            `json:"user"`
	Directory  string            `json:"directory"`
	Shell      string            `json:"shell"`
	Terminal   string            `json:"terminal"`
	System     string            `json:"system"`
	Platform   string            `json:"platform"`
	IsGitRepo  bool              `json:"is_git_repo"`
	GitBranch  string            `json:"git_branch,omitempty"`
	DateTime   time.Time         `json:"datetime"`
	Aliases    map[string]string `json:"aliases"`
	K8sContext *K8sContext       `json:"k8s_context,omitempty"`
}

// K8sContext contains Kubernetes environment information
type K8sContext struct {
	IsAvailable      bool   `json:"is_available"`
	CurrentContext   string `json:"current_context,omitempty"`
	CurrentNamespace string `json:"current_namespace,omitempty"`
	ClusterInfo      string `json:"cluster_info,omitempty"`
}

// HistoryEntry represents a shell command and its result
type HistoryEntry struct {
	Command     string    `json:"command"`
	Output      string    `json:"output"`
	ErrorOutput string    `json:"error_output"`
	ExitCode    int       `json:"exit_code"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    string    `json:"duration,omitempty"`
}

// Response represents the AI's response
type Response struct {
	Type    ResponseType `json:"type"`
	Content string       `json:"content"`
}

// ResponseType indicates the type of AI response
type ResponseType string

const (
	TypeCompletion  ResponseType = "completion"  // prefix with +
	TypeReplacement ResponseType = "replacement" // prefix with =
	TypePrediction  ResponseType = "prediction"  // new command suggestion
)
