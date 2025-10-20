# AutoCoder RAG Go SDK 使用手册

## 📚 目录

1. [快速入门](#快速入门)
2. [安装指南](#安装指南)
3. [基础使用](#基础使用)
4. [高级功能](#高级功能)
5. [配置参数详解](#配置参数详解)
6. [API参考](#api参考)
7. [错误处理](#错误处理)
8. [最佳实践](#最佳实践)
9. [故障排查](#故障排查)

---

## 快速入门

### 30秒快速体验

```go
package main

import (
    "fmt"
    "log"
    
    ragclient "github.com/autocoder/rag-sdk-go"
)

func main() {
    // 3行代码开始
    client, _ := ragclient.NewRAGClient("/path/to/docs")
    answer, err := client.Query("如何使用这个项目?", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(answer)
}
```

**说明**: Go SDK使用工厂函数模式，符合Go语言惯用法。

### 3分钟完整示例

```go
package main

import (
    "fmt"
    "log"
    
    ragclient "github.com/autocoder/rag-sdk-go"
)

func main() {
    // 创建配置
    config := ragclient.NewRAGConfig("/path/to/docs")
    config.Model = "v3_chat"
    config.Timeout = 600  // 10分钟超时
    config.Agentic = true
    config.ProductMode = "pro"

    // 创建客户端
    client, err := ragclient.NewRAGClientWithConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    // 方式1: 标准查询
    answer, err := client.Query("这个项目是做什么的?", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("答案: %s\n\n", answer)

    // 方式2: 流式输出（Channel接口）
    fmt.Print("流式答案: ")
    resultChan, errorChan := client.QueryStream("有哪些主要功能?", nil)
    
    for {
        select {
        case line, ok := <-resultChan:
            if !ok {
                fmt.Println()
                goto next
            }
            fmt.Print(line)
            
        case err := <-errorChan:
            if err != nil {
                log.Printf("错误: %v\n", err)
                return
            }
        }
    }

next:
    // 方式3: 使用工具函数收集结果
    resultChan2, errorChan2 := client.QueryStream("如何安装?", nil)
    fullAnswer, err := ragclient.QueryWithBuffer(resultChan2, errorChan2)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\n完整答案: %s\n", fullAnswer)
}
```

---

## 安装指南

### 系统要求

- Go 1.21+
- `auto-coder.rag` 命令已安装

### 安装SDK

```bash
go get github.com/autocoder/rag-sdk-go
```

### 验证安装

```go
package main

import (
    "fmt"
    ragclient "github.com/autocoder/rag-sdk-go"
)

func main() {
    client, _ := ragclient.NewRAGClient(".")
    
    fmt.Printf("命令可用: %v\n", client.CheckAvailability())
    fmt.Printf("版本: %s\n", client.GetVersion())
}
```

---

## 基础使用

### 1. 创建客户端

Go SDK 使用工厂函数模式，提供两种清晰的创建方式：

#### 方式1: 使用字符串 - 最简单（推荐快速开始）

```go
client, err := ragclient.NewRAGClient("/path/to/docs")
if err != nil {
    log.Fatal(err)
}
```

**适用场景**: 快速开始、使用默认配置

**说明**: `NewRAGClient()` 使用默认配置创建客户端。

#### 方式2: 使用配置对象（推荐需要自定义配置）⭐

```go
config := ragclient.NewRAGConfig("/path/to/docs")
config.Model = "v3_chat"
config.Timeout = 600  // 10分钟（秒）
config.Agentic = true
config.ProductMode = "pro"

client, err := ragclient.NewRAGClientWithConfig(config)
if err != nil {
    log.Fatal(err)
}
```

**适用场景**: 需要自定义配置

**优点**:
- ✅ 配置对象可复用
- ✅ 字段直接赋值
- ✅ 类型安全

**说明**: Go SDK使用两个工厂函数，符合Go惯用法。

### 2. 执行查询

#### 标准查询

```go
answer, err := client.Query("如何配置?", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println(answer)
```

#### 带选项的查询

```go
timeout := 900  // 15分钟
options := &ragclient.RAGQueryOptions{
    OutputFormat: "text",
    Timeout:      &timeout,
}

answer, err := client.Query("复杂问题", options)
if err != nil {
    log.Fatal(err)
}
fmt.Println(answer)
```

#### 流式查询（Channel接口）

```go
resultChan, errorChan := client.QueryStream("详细说明", nil)

for {
    select {
    case line, ok := <-resultChan:
        if !ok {
            return
        }
        fmt.Println(line)
        
    case err := <-errorChan:
        if err != nil {
            log.Printf("错误: %v", err)
            return
        }
    }
}
```

#### 使用工具函数收集流式结果

```go
resultChan, errorChan := client.QueryStream("问题", nil)

// 自动收集所有输出到字符串
answer, err := ragclient.QueryWithBuffer(resultChan, errorChan)
if err != nil {
    log.Fatal(err)
}
fmt.Println(answer)
```

---

## 高级功能

### 1. 超时配置

#### 全局超时

```go
config := ragclient.NewRAGConfig("/path/to/docs")
config.Timeout = 600  // 10分钟
client, _ := ragclient.NewRAGClientWithConfig(config)
```

#### 查询级超时

```go
timeout := 900  // 15分钟
options := &ragclient.RAGQueryOptions{
    Timeout: &timeout,
}

answer, _ := client.Query("深度分析", options)
```

### 2. 混合索引

```go
config := ragclient.NewRAGConfig("/path/to/large_docs")
config.EnableHybridIndex = true
config.RagContextWindowLimit = 100000

client, _ := ragclient.NewRAGClientWithConfig(config)
```

### 3. 并发查询

```go
import "sync"

func batchQuery(client *ragclient.RAGClient, questions []string) map[string]string {
    results := make(map[string]string)
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, question := range questions {
        wg.Add(1)
        go func(q string) {
            defer wg.Done()
            
            answer, err := client.Query(q, nil)
            mu.Lock()
            defer mu.Unlock()
            
            if err != nil {
                results[q] = fmt.Sprintf("错误: %v", err)
            } else {
                results[q] = answer
            }
        }(question)
    }

    wg.Wait()
    return results
}
```

---

## 配置参数详解

### RAGConfig 字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `DocDir` | string | **必需** | 文档目录 |
| `Model` | string | "v3_chat" | 模型名称 |
| `Timeout` | int | 300 | 超时（秒） |
| `Agentic` | bool | false | AgenticRAG |
| `ProductMode` | string | "lite" | 产品模式 |
| `RagContextWindowLimit` | int | 56000 | 上下文窗口 |
| `FullTextRatio` | float64 | 0.7 | 全文比例 |
| `SegmentRatio` | float64 | 0.2 | 片段比例 |
| `EnableHybridIndex` | bool | false | 混合索引 |

### RAGQueryOptions 字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `OutputFormat` | string | "text" | 输出格式 |
| `Agentic` | *bool | nil | 覆盖RAG模式 |
| `ProductMode` | string | "" | 覆盖产品模式 |
| `Model` | string | "" | 覆盖模型 |
| `Timeout` | *int | nil | 覆盖超时（秒） |

---

## 错误处理

### 错误类型

```go
import ragclient "github.com/autocoder/rag-sdk-go"

answer, err := client.Query("问题", nil)
if err != nil {
    switch e := err.(type) {
    case *ragclient.ValidationError:
        fmt.Printf("参数验证错误: %s\n", e.Message)
        
    case *ragclient.ExecutionError:
        fmt.Printf("执行错误: %s\n", e.Message)
        fmt.Printf("退出码: %d\n", e.ExitCode)
        
    case *ragclient.RAGError:
        fmt.Printf("SDK错误: %s\n", e.Message)
        
    default:
        fmt.Printf("未知错误: %v\n", err)
    }
}
```

---

## 最佳实践

### 1. 全局客户端

```go
package myapp

import ragclient "github.com/autocoder/rag-sdk-go"

var globalClient *ragclient.RAGClient

func InitRAGClient(docDir string) error {
    config := ragclient.NewRAGConfig(docDir)
    config.ProductMode = "pro"
    config.Timeout = 600
    
    client, err := ragclient.NewRAGClientWithConfig(config)
    if err != nil {
        return err
    }
    
    globalClient = client
    return nil
}

func Ask(question string) (string, error) {
    return globalClient.Query(question, nil)
}
```

### 2. 封装为服务

```go
type DocQAService struct {
    client *ragclient.RAGClient
}

func NewDocQAService(docDir string) (*DocQAService, error) {
    config := ragclient.NewRAGConfig(docDir)
    config.Agentic = true
    config.ProductMode = "pro"
    
    client, err := ragclient.NewRAGClientWithConfig(config)
    if err != nil {
        return nil, err
    }
    
    return &DocQAService{client: client}, nil
}

func (s *DocQAService) Ask(question string) (string, error) {
    return s.client.Query(question, nil)
}

func (s *DocQAService) StreamAnswer(question string) (<-chan string, <-chan error) {
    return s.client.QueryStream(question, nil)
}
```

---

## 相关资源

- [Go SDK README](./README.md)
- [FAQ文档](../FAQ.md)
- [最佳实践](../BEST_PRACTICES.md)
- [示例代码](./examples/)

---

**版本**: 1.0.0  
**Go版本要求**: 1.21+  
**最后更新**: 2025-10-19

