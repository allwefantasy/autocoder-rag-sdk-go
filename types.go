package ragclient

import (
	"encoding/json"
	"errors"
	"time"
)

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

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeStart    MessageType = "start"
	MessageTypeStage    MessageType = "stage"
	MessageTypeContent  MessageType = "content"
	MessageTypeContexts MessageType = "contexts"
	MessageTypeEnd      MessageType = "end"
)

// StageType represents the type of processing stage
type StageType string

const (
	StageTypeProcessing StageType = "processing"
	StageTypeRetrieval  StageType = "retrieval"
	StageTypeFiltering  StageType = "filtering"
	StageTypeChunking   StageType = "chunking"
	StageTypeGeneration StageType = "generation"
)

// TokenInfo represents token information
type TokenInfo struct {
	Input     int `json:"input"`
	Generated int `json:"generated"`
}

// Message represents a unified RAG message object
type Message struct {
	EventType MessageType         `json:"event_type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	RawJSON   string             `json:"-"` // Original JSON string for debugging
}

// FromJSON creates a Message object from JSON string
func (m *Message) FromJSON(jsonStr string) error {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return err
	}

	// Parse timestamp
	timestampStr, ok := data["timestamp"].(string)
	if !ok {
		m.Timestamp = time.Now()
	} else {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			m.Timestamp = timestamp
		} else {
			m.Timestamp = time.Now()
		}
	}

	// Parse event type
	eventTypeStr, ok := data["event_type"].(string)
	if !ok {
		return errors.New("missing event_type")
	}
	m.EventType = MessageType(eventTypeStr)

	// Parse data
	if dataMap, ok := data["data"].(map[string]interface{}); ok {
		m.Data = dataMap
	} else {
		m.Data = make(map[string]interface{})
	}

	m.RawJSON = jsonStr
	return nil
}

// ToJSON converts Message to JSON string
func (m *Message) ToJSON() (string, error) {
	data := map[string]interface{}{
		"event_type": string(m.EventType),
		"timestamp":  m.Timestamp.Format(time.RFC3339),
		"data":       m.Data,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// Convenience methods for checking message type
func (m *Message) IsStart() bool {
	return m.EventType == MessageTypeStart
}

func (m *Message) IsStage() bool {
	return m.EventType == MessageTypeStage
}

func (m *Message) IsContent() bool {
	return m.EventType == MessageTypeContent
}

func (m *Message) IsContexts() bool {
	return m.EventType == MessageTypeContexts
}

func (m *Message) IsEnd() bool {
	return m.EventType == MessageTypeEnd
}

// Convenience methods for getting specific data
func (m *Message) GetStatus() string {
	if status, ok := m.Data["status"].(string); ok {
		return status
	}
	return ""
}

func (m *Message) GetStageType() StageType {
	if stageTypeStr, ok := m.Data["type"].(string); ok {
		return StageType(stageTypeStr)
	}
	return ""
}

func (m *Message) GetMessage() string {
	if message, ok := m.Data["message"].(string); ok {
		return message
	}
	return ""
}

func (m *Message) GetContent() string {
	if content, ok := m.Data["content"].(string); ok {
		return content
	}
	return ""
}

func (m *Message) GetContexts() []string {
	if contexts, ok := m.Data["contexts"].([]interface{}); ok {
		result := make([]string, len(contexts))
		for i, ctx := range contexts {
			if str, ok := ctx.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	return nil
}

func (m *Message) GetTokens() *TokenInfo {
	if tokensData, ok := m.Data["tokens"].(map[string]interface{}); ok {
		input, _ := tokensData["input"].(float64)
		generated, _ := tokensData["generated"].(float64)
		return &TokenInfo{
			Input:     int(input),
			Generated: int(generated),
		}
	}
	return nil
}

func (m *Message) GetMetadata() map[string]interface{} {
	if metadata, ok := m.Data["metadata"].(map[string]interface{}); ok {
		return metadata
	}
	return nil
}

// Convenience methods for checking stage type
func (m *Message) IsProcessingStage() bool {
	return m.IsStage() && m.GetStageType() == StageTypeProcessing
}

func (m *Message) IsRetrievalStage() bool {
	return m.IsStage() && m.GetStageType() == StageTypeRetrieval
}

func (m *Message) IsFilteringStage() bool {
	return m.IsStage() && m.GetStageType() == StageTypeFiltering
}

func (m *Message) IsChunkingStage() bool {
	return m.IsStage() && m.GetStageType() == StageTypeChunking
}

func (m *Message) IsGenerationStage() bool {
	return m.IsStage() && m.GetStageType() == StageTypeGeneration
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

