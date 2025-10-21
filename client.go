package ragclient

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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

		// 启动命令
		if err := execCmd.Start(); err != nil {
			errorChan <- &RAGError{Message: fmt.Sprintf("启动命令失败: %v", err)}
			return
		}

		// 写入问题
		go func() {
			defer stdin.Close()
			stdin.Write([]byte(question))
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
	cmd := []string{"auto-coder.rag", "run", "--doc_dir", c.config.DocDir}

	// 模型参数
	model := c.config.Model
	if options.Model != "" {
		model = options.Model
	}
	if model != "" {
		cmd = append(cmd, "--model", model)
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
		"--rag_doc_filter_relevance", strconv.Itoa(c.config.RagDocFilterRelevance),
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

	cmd := exec.CommandContext(ctx, "auto-coder.rag", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// CheckAvailability checks if auto-coder.rag command is available
func (c *RAGClient) CheckAvailability() bool {
	// Check if command exists
	if _, err := exec.LookPath("auto-coder.rag"); err != nil {
		return false
	}

	// Test if command works
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)  // 60秒超时
	defer cancel()

	cmd := exec.CommandContext(ctx, "auto-coder.rag", "--help")
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

