package ragclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RAGClient is the main client for interacting with auto-coder.rag run
type RAGClient struct {
	config *RAGConfig
}

// NewRAGClient creates a new RAG client
func NewRAGClient(docDir string) (*RAGClient, error) {
	return NewRAGClientWithConfig(NewRAGConfig(docDir))
}

// NewRAGClientWithConfig creates a new RAG client with custom config
func NewRAGClientWithConfig(config *RAGConfig) (*RAGClient, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return &RAGClient{
		config: config,
	}, nil
}

// GetDocDir returns the document directory path
//
// Useful for debugging or manual cleanup of temp directories.
func (c *RAGClient) GetDocDir() string {
	return c.config.DocDir
}

// NewRAGClientFromText creates a RAG client from text content
//
// Creates a temporary directory with the text content as a document file,
// then initializes a client based on that directory.
// The temporary directory will NOT be automatically cleaned up.
// Use client.GetDocDir() to get the path for manual cleanup.
//
// Example:
//
//	client, err := ragclient.NewRAGClientFromText("This is document content...", "", "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	answer, err := client.Query("Question?", nil)
//
//	// Get temp directory path (for debugging or manual cleanup)
//	fmt.Println("Doc directory:", client.GetDocDir())
//
//	// Manual cleanup
//	os.RemoveAll(client.GetDocDir())
func NewRAGClientFromText(text string, filename string, tempDir string) (*RAGClient, error) {
	// Validate text content
	if strings.TrimSpace(text) == "" {
		return nil, &ValidationError{Message: "Text content cannot be empty"}
	}

	if filename == "" {
		filename = "document.md"
	}

	// Create directory
	var docPath string
	var err error
	if tempDir != "" {
		docPath = tempDir
		if err := os.MkdirAll(docPath, 0755); err != nil {
			return nil, &RAGError{Message: fmt.Sprintf("Failed to create directory: %v", err)}
		}
	} else {
		docPath, err = os.MkdirTemp("", "rag_text_")
		if err != nil {
			return nil, &RAGError{Message: fmt.Sprintf("Failed to create temp directory: %v", err)}
		}
	}

	// Write file
	filePath := docPath + "/" + filename
	if err := os.WriteFile(filePath, []byte(text), 0644); err != nil {
		return nil, &RAGError{Message: fmt.Sprintf("Failed to write file: %v", err)}
	}

	// Create and return client
	return NewRAGClient(docPath)
}

// NewRAGClientFromTexts creates a RAG client from multiple text documents
//
// Creates a temporary directory with multiple document files,
// then initializes a client based on that directory.
// The temporary directory will NOT be automatically cleaned up.
// Use client.GetDocDir() to get the path for manual cleanup.
//
// Example:
//
//	docs := []ragclient.TextDocument{
//	    {Content: "API documentation...", Filename: "api.md"},
//	    {Content: "User guide...", Filename: "guide.md"},
//	    {Content: "FAQ...", Filename: "faq.md"},
//	}
//
//	client, err := ragclient.NewRAGClientFromTexts(docs, "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	answer, err := client.Query("How to use the API?", nil)
//
//	// Get temp directory path
//	fmt.Println("Doc directory:", client.GetDocDir())
func NewRAGClientFromTexts(texts []TextDocument, tempDir string) (*RAGClient, error) {
	// Validate documents
	if len(texts) == 0 {
		return nil, &ValidationError{Message: "At least one document is required"}
	}

	for _, doc := range texts {
		if strings.TrimSpace(doc.Content) == "" {
			filename := doc.Filename
			if filename == "" {
				filename = "unknown"
			}
			return nil, &ValidationError{Message: fmt.Sprintf("Document '%s' content cannot be empty", filename)}
		}
	}

	// Create directory
	var docPath string
	var err error
	if tempDir != "" {
		docPath = tempDir
		if err := os.MkdirAll(docPath, 0755); err != nil {
			return nil, &RAGError{Message: fmt.Sprintf("Failed to create directory: %v", err)}
		}
	} else {
		docPath, err = os.MkdirTemp("", "rag_texts_")
		if err != nil {
			return nil, &RAGError{Message: fmt.Sprintf("Failed to create temp directory: %v", err)}
		}
	}

	// Write all files
	for i, doc := range texts {
		filename := doc.Filename
		if filename == "" {
			filename = fmt.Sprintf("doc_%d.md", i)
		}

		filePath := docPath + "/" + filename
		if err := os.WriteFile(filePath, []byte(doc.Content), 0644); err != nil {
			return nil, &RAGError{Message: fmt.Sprintf("Failed to write file %s: %v", filename, err)}
		}
	}

	// Create and return client
	return NewRAGClient(docPath)
}

func validateConfig(config *RAGConfig) error {
	// 验证文档目录存在
	if _, err := os.Stat(config.DocDir); os.IsNotExist(err) {
		return &ValidationError{Message: fmt.Sprintf("文档目录不存在: %s", config.DocDir)}
	}

	// 验证产品模式
	if config.ProductMode != "lite" && config.ProductMode != "pro" {
		return &ValidationError{Message: fmt.Sprintf("不支持的产品模式: %s", config.ProductMode)}
	}

	return nil
}

// buildEnv builds the environment variables for subprocess
// Merge priority (low to high):
// 1. os.Environ (system environment)
// 2. config.WindowsUtf8Env (Windows UTF-8 auto-config)
// 3. config.Envs (global config)
// 4. options.Envs (single query config)
func (c *RAGClient) buildEnv(options *RAGQueryOptions) []string {
	env := os.Environ()
	envMap := make(map[string]string)

	// 1. Parse system environment
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// 2. Windows UTF-8 auto-config
	if c.config.WindowsUtf8Env && runtime.GOOS == "windows" {
		envMap["PYTHONIOENCODING"] = "utf-8"
		envMap["LANG"] = "zh_CN.UTF-8"
		envMap["LC_ALL"] = "zh_CN.UTF-8"
		envMap["CHCP"] = "65001"
	}

	// 3. Global config
	if c.config.Envs != nil {
		for k, v := range c.config.Envs {
			envMap[k] = v
		}
	}

	// 4. Single query config (highest priority)
	if options != nil && options.Envs != nil {
		for k, v := range options.Envs {
			envMap[k] = v
		}
	}

	// Convert back to []string
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}

// Query executes a RAG query and returns the complete answer
func (c *RAGClient) Query(question string, options *RAGQueryOptions) (string, error) {
	if options == nil {
		options = &RAGQueryOptions{OutputFormat: "text"}
	}

	// 验证输出格式
	if options.OutputFormat != "text" && options.OutputFormat != "json" && options.OutputFormat != "stream-json" {
		return "", &ValidationError{Message: fmt.Sprintf("不支持的输出格式: %s", options.OutputFormat)}
	}

	// 获取超时时间
	timeout := c.config.Timeout
	if options.Timeout != nil {
		timeout = *options.Timeout
	}

	cmd := c.buildCommand(options)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	execCmd.Env = c.buildEnv(options)

	// 设置输入
	execCmd.Stdin = strings.NewReader(question)

	// 执行命令
	output, err := execCmd.CombinedOutput()

	if err != nil {
		if execCmd.ProcessState != nil {
			exitCode := execCmd.ProcessState.ExitCode()
			return "", &ExecutionError{
				Message:  string(output),
				ExitCode: exitCode,
			}
		}
		return "", &RAGError{Message: fmt.Sprintf("执行查询时发生错误: %v", err)}
	}

	return strings.TrimSpace(string(output)), nil
}

// QueryStream executes a RAG query and streams the results
func (c *RAGClient) QueryStream(question string, options *RAGQueryOptions) (<-chan string, <-chan error) {
	resultChan := make(chan string, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		if options == nil {
			options = &RAGQueryOptions{OutputFormat: "text"}
		}

		cmd := c.buildCommand(options)

		execCmd := exec.Command(cmd[0], cmd[1:]...)
		execCmd.Env = c.buildEnv(options)

		// 设置输入
		stdin, err := execCmd.StdinPipe()
		if err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("创建stdin失败: %v", err)}
			return
		}

		// 设置输出
		stdout, err := execCmd.StdoutPipe()
		if err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("创建stdout失败: %v", err)}
			return
		}

		// 设置错误输出
		stderr, err := execCmd.StderrPipe()
		if err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("创建stderr失败: %v", err)}
			return
		}

		// 启动命令
		if err := execCmd.Start(); err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("启动命令失败: %v (命令: %s)", err, cmd[0])}
			return
		}

		// 写入问题
		go func() {
			defer stdin.Close()
			stdin.Write([]byte(question))
		}()

		// 异步读取 stderr
		var stderrOutput []byte
		go func() {
			stderrOutput, _ = io.ReadAll(stderr)
		}()

		// 流式读取输出
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			resultChan <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("读取输出失败: %v", err)}
			return
		}

		// 等待命令完成
		if err := execCmd.Wait(); err != nil {
			stderrStr := strings.TrimSpace(string(stderrOutput))
			if execCmd.ProcessState != nil {
				exitCode := execCmd.ProcessState.ExitCode()
				errMsg := fmt.Sprintf("命令执行失败 (退出码: %d, 命令: %s)", exitCode, cmd[0])
				if stderrStr != "" {
					errMsg += fmt.Sprintf("\n错误输出: %s", stderrStr)
				}
				errorChan <- &ExecutionError{
					Message:  errMsg,
					ExitCode: exitCode,
				}
			} else {
				errMsg := fmt.Sprintf("命令执行失败: %v (命令: %s)", err, cmd[0])
				if stderrStr != "" {
					errMsg += fmt.Sprintf("\n错误输出: %s", stderrStr)
				}
				errorChan <- &RAGError{Message: errMsg}
			}
		}
	}()

	return resultChan, errorChan
}

// QueryStreamMessages executes a RAG query and returns Message objects stream
func (c *RAGClient) QueryStreamMessages(question string, options *RAGQueryOptions) (<-chan *Message, <-chan error) {
	messageChan := make(chan *Message, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(messageChan)
		defer close(errorChan)

		if options == nil {
			options = &RAGQueryOptions{OutputFormat: "stream-json"}
		} else {
			// Ensure using stream-json format
			options = &RAGQueryOptions{
				OutputFormat: "stream-json",
				Agentic:      options.Agentic,
				ProductMode:  options.ProductMode,
				Model:        options.Model,
				Timeout:      options.Timeout,
			}
		}

		cmd := c.buildCommand(options)

		execCmd := exec.Command(cmd[0], cmd[1:]...)
		execCmd.Env = c.buildEnv(options)

		// Set up input
		stdin, err := execCmd.StdinPipe()
		if err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("创建stdin失败: %v", err)}
			return
		}

		// Set up output
		stdout, err := execCmd.StdoutPipe()
		if err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("创建stdout失败: %v", err)}
			return
		}

		// Start command
		if err := execCmd.Start(); err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("启动命令失败: %v", err)}
			return
		}

		// Write question
		go func() {
			defer stdin.Close()
			stdin.Write([]byte(question))
		}()

		// Stream read output
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			message := &Message{}
			if err := message.FromJSON(line); err != nil {
				// Skip invalid JSON lines
				continue
			}
			messageChan <- message
		}

		if err := scanner.Err(); err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("读取输出失败: %v", err)}
			return
		}

		// Wait for command completion
		if err := execCmd.Wait(); err != nil {
			if execCmd.ProcessState != nil {
				errorChan <- &ExecutionError{
					Message:  fmt.Sprintf("命令执行失败"),
					ExitCode: execCmd.ProcessState.ExitCode(),
				}
			} else {
				errorChan <- &RAGError{Message: fmt.Sprintf("命令执行失败: %v", err)}
			}
		}
	}()

	return messageChan, errorChan
}

// QueryCollectMessages executes a query and returns a RAGResponse with Message stream
func (c *RAGClient) QueryCollectMessages(question string, options *RAGQueryOptions) (*RAGResponse, error) {
	var contentParts []string
	var contexts []string
	var metadata map[string]interface{}
	tokensInfo := map[string]int{"input": 0, "generated": 0}

	messageChan, errorChan := c.QueryStreamMessages(question, options)

	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				// Channel closed, build response
				answer := strings.Join(contentParts, "")
				
				// Add token info to metadata
				if metadata == nil {
					metadata = make(map[string]interface{})
				}
				metadata["tokens"] = tokensInfo

				return &RAGResponse{
					Success:  true,
					Answer:   answer,
					Contexts: contexts,
					Error:    "",
				}, nil
			}

			if message.IsContent() {
				contentParts = append(contentParts, message.GetContent())
			} else if message.IsContexts() {
				contexts = append(contexts, message.GetContexts()...)
			} else if message.IsEnd() {
				metadata = message.GetMetadata()
			} else if tokens := message.GetTokens(); tokens != nil {
				tokensInfo["input"] += tokens.Input
				tokensInfo["generated"] += tokens.Generated
			}

		case err, ok := <-errorChan:
			if ok && err != nil {
				return &RAGResponse{
					Success: false,
					Answer:  "",
					Error:   err.Error(),
				}, err
			}
		}
	}
}

func (c *RAGClient) buildCommand(options *RAGQueryOptions) []string {
	cmd := []string{c.config.CommandPath, "run", "--doc_dir", c.config.DocDir}

	// 模型参数
	model := c.config.Model
	if options.Model != "" {
		model = options.Model
	}
	if model != "" {
		cmd = append(cmd, "--model", model)
	}

	// 模型配置文件
	modelFile := c.config.ModelFile
	if options.ModelFile != "" {
		modelFile = options.ModelFile
	}
	if modelFile != "" {
		cmd = append(cmd, "--model_file", modelFile)
	}

	// 输出格式
	outputFormat := options.OutputFormat
	if outputFormat == "" {
		outputFormat = "text"
	}
	cmd = append(cmd, "--output_format", outputFormat)

	// RAG 模式
	agentic := c.config.Agentic
	if options.Agentic != nil {
		agentic = *options.Agentic
	}
	if agentic {
		cmd = append(cmd, "--agentic")
	}

	// 产品模式
	productMode := c.config.ProductMode
	if options.ProductMode != "" {
		productMode = options.ProductMode
	}
	if productMode == "pro" {
		cmd = append(cmd, "--pro")
	} else if productMode == "lite" {
		cmd = append(cmd, "--lite")
	}

	// RAG 参数
	cmd = append(cmd,
		"--rag_context_window_limit", strconv.Itoa(c.config.RagContextWindowLimit),
		"--full_text_ratio", fmt.Sprintf("%.1f", c.config.FullTextRatio),
		"--segment_ratio", fmt.Sprintf("%.1f", c.config.SegmentRatio),
		"--rag_doc_filter_relevance", fmt.Sprintf("%d", int(c.config.RagDocFilterRelevance)),
	)

	// 索引选项
	if c.config.EnableHybridIndex {
		cmd = append(cmd, "--enable_hybrid_index")
	}
	if c.config.DisableAutoWindow {
		cmd = append(cmd, "--disable_auto_window")
	}
	if c.config.DisableSegmentReorder {
		cmd = append(cmd, "--disable_segment_reorder")
	}

	return cmd
}

// GetVersion returns the auto-coder.rag version
func (c *RAGClient) GetVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)  // 60秒超时
	defer cancel()

	cmd := exec.CommandContext(ctx, c.config.CommandPath, "--version")
	cmd.Env = c.buildEnv(nil)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// CheckAvailability checks if auto-coder.rag command is available
func (c *RAGClient) CheckAvailability() bool {
	// Test if command works
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)  // 60秒超时
	defer cancel()

	cmd := exec.CommandContext(ctx, c.config.CommandPath, "--help")
	cmd.Env = c.buildEnv(nil)
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// QueryWithBuffer is a helper that collects stream results into a string
func QueryWithBuffer(resultChan <-chan string, errorChan <-chan error) (string, error) {
	var buffer bytes.Buffer

	for {
		select {
		case line, ok := <-resultChan:
			if !ok {
				return buffer.String(), nil
			}
			buffer.WriteString(line)
			buffer.WriteString("\n")

		case err, ok := <-errorChan:
			if ok && err != nil {
				return buffer.String(), err
			}
		}
	}
}

// CountTokens counts tokens in a file
//
// This is a standalone function that can be called without creating a client instance.
// It calls `auto-coder.rag tools count --file <file_path> --output_format json` to count tokens.
//
// Example:
//
//	result, err := ragclient.CountTokens("/path/to/file.xlsx", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Total tokens: %d\n", result.TotalTokens)
//
//	// With options
//	result, err := ragclient.CountTokens("/path/to/file.xlsx", &ragclient.TokenCountOptions{
//	    TokenizerPath: "/path/to/tokenizer.json",
//	})
func CountTokens(filePath string, options *TokenCountOptions) (*TokenCountResult, error) {
	if options == nil {
		options = &TokenCountOptions{Timeout: 60}
	}
	if options.Timeout == 0 {
		options.Timeout = 60
	}

	commandPath := "auto-coder.rag"

	// Build command - always use JSON output format
	cmd := []string{commandPath, "tools", "count", "--file", filePath, "--output_format", "json"}

	if options.TokenizerPath != "" {
		cmd = append(cmd, "--tokenizer_path", options.TokenizerPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(options.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)

	// Build environment
	env := os.Environ()
	if options.Envs != nil {
		for k, v := range options.Envs {
			env = append(env, k+"="+v)
		}
	}
	execCmd.Env = env

	// Execute command
	output, err := execCmd.CombinedOutput()

	if err != nil {
		if execCmd.ProcessState != nil {
			exitCode := execCmd.ProcessState.ExitCode()
			return nil, &ExecutionError{
				Message:  string(output),
				ExitCode: exitCode,
			}
		}
		return nil, &RAGError{Message: fmt.Sprintf("Error executing token count: %v", err)}
	}

	// Parse JSON output
	return parseTokenCountJsonOutput(strings.TrimSpace(string(output)))
}

// parseTokenCountJsonOutput parses the JSON output from the token count command
func parseTokenCountJsonOutput(output string) (*TokenCountResult, error) {
	var result struct {
		Files           []TokenCountFileResult `json:"files"`
		TotalCharacters int                    `json:"totalCharacters"`
		TotalTokens     int                    `json:"totalTokens"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, &RAGError{Message: fmt.Sprintf("Failed to parse JSON output: %v. Output was: %s", err, output[:min(200, len(output))])}
	}

	return &TokenCountResult{
		Files:           result.Files,
		TotalCharacters: result.TotalCharacters,
		TotalTokens:     result.TotalTokens,
		RawOutput:       output,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

