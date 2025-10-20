package main

import (
	"fmt"
	"log"

	ragclient "github.com/autocoder/rag-sdk-go"
)

func main() {
	fmt.Println("=== AutoCoder RAG SDK Go 基础用法演示 ===\n")

	// 1. 创建客户端
	fmt.Println("1. 创建客户端...")
	client, err := ragclient.NewRAGClient(".")
	if err != nil {
		log.Fatalf("❌ 客户端创建失败: %v", err)
	}
	fmt.Println("✅ 客户端创建成功")

	// 2. 检查命令可用性
	fmt.Println("\n2. 检查 auto-coder.rag 命令...")
	if client.CheckAvailability() {
		fmt.Println("   ✅ auto-coder.rag 命令可用")
		version := client.GetVersion()
		fmt.Printf("   版本: %s\n", version)
	} else {
		fmt.Println("   ❌ auto-coder.rag 命令不可用")
		fmt.Println("   请确保已安装 auto-coder 并且命令在 PATH 中")
		return
	}

	// 3. 基础查询
	fmt.Println("\n3. 基础查询示例...")
	question := "这个目录下有哪些 Go 文件?"

	fmt.Printf("   问题: %s\n", question)
	answer, err := client.Query(question, nil)
	if err != nil {
		fmt.Printf("   ❌ 查询失败: %v\n", err)
	} else {
		preview := answer
		if len(answer) > 200 {
			preview = answer[:200] + "..."
		}
		fmt.Printf("   答案: %s\n", preview)
	}

	fmt.Println("\n=== 基础用法演示完成 ===")
}

