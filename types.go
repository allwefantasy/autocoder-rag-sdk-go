package ragclient

import "errors"

// RAGConfig represents the RAG configuration
type RAGConfig struct {
	DocDir string

	// Model configuration
	Model string
	
	// Timeout configuration (seconds)
	Timeout int

	// RAG parameters
	RagContextWindowLimit  int
	FullTextRatio          float64
	SegmentRatio           float64
	RagDocFilterRelevance  int

	// Mode selection
	Agentic     bool
	ProductMode string // "lite" or "pro"

	// Index configuration
	EnableHybridIndex      bool
	DisableAutoWindow      bool
	DisableSegmentReorder  bool

	// Optional model configuration
	RecallModel        string
	ChunkModel         string
	QAModel            string
	EmbModel           string
	AgenticModel       string
	ContextPruneModel  string

	// Tokenizer path
	TokenizerPath string

	// Other parameters
	RequiredExts string
	RayAddress   string
}

// NewRAGConfig creates a new RAG configuration with defaults
func NewRAGConfig(docDir string) *RAGConfig {
	return &RAGConfig{
		DocDir:                 docDir,
		Model:                  "v3_chat",
		Timeout:                300,  // 默认5分钟
		RagContextWindowLimit:  56000,
		FullTextRatio:          0.7,
		SegmentRatio:           0.2,
		RagDocFilterRelevance:  5,
		Agentic:                false,
		ProductMode:            "lite",
		EnableHybridIndex:      false,
		DisableAutoWindow:      false,
		DisableSegmentReorder:  false,
		RayAddress:             "auto",
	}
}

// RAGQueryOptions represents options for a single query
type RAGQueryOptions struct {
	OutputFormat string // "text", "json", or "stream-json"
	Agentic      *bool
	ProductMode  string
	Model        string
	Timeout      *int   // Timeout in seconds (overrides config)
}

// RAGResponse represents a RAG query response
type RAGResponse struct {
	Success  bool
	Answer   string
	Contexts []string
	Error    string
}

// Custom error types
var (
	ErrValidation = errors.New("参数验证失败")
	ErrExecution  = errors.New("执行失败")
)

// RAGError represents a general SDK error
type RAGError struct {
	Message string
}

func (e *RAGError) Error() string {
	return e.Message
}

// ValidationError represents parameter validation errors
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ExecutionError represents execution errors
type ExecutionError struct {
	Message  string
	ExitCode int
}

func (e *ExecutionError) Error() string {
	return e.Message
}

