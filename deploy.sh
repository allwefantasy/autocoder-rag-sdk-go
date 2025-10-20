#!/bin/bash
# AutoCoder RAG Go SDK 发布脚本
# 
# Go modules 通过 Git tags 发布，本脚本用于辅助发布流程
#
# 使用方法:
#   ./deploy.sh        # 显示发布指南
#   ./deploy.sh check  # 检查发布准备情况

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 检查是否在正确的目录
if [ ! -f "go.mod" ]; then
    print_error "请在 rag-sdks/go 目录下运行此脚本"
    exit 1
fi

MODE=${1:-"guide"}

print_info "========================================="
print_info "  AutoCoder RAG Go SDK 发布助手"
print_info "========================================="

# 读取 module 信息
MODULE_PATH=$(grep '^module ' go.mod | awk '{print $2}')
print_info "Module 路径: ${MODULE_PATH}"

if [ "$MODE" = "check" ]; then
    print_info ""
    print_info "检查发布准备情况..."
    echo ""
    
    # 1. 检查 Go 环境
    print_step "1. 检查 Go 环境"
    if ! command -v go &> /dev/null; then
        print_error "   未找到 go，请先安装 Go"
        exit 1
    fi
    GO_VERSION=$(go version)
    print_info "   ✓ Go 已安装: ${GO_VERSION}"
    
    # 2. 检查代码编译
    print_step "2. 检查代码编译"
    if go build ./...; then
        print_info "   ✓ 代码编译通过"
    else
        print_error "   ✗ 代码编译失败"
        exit 1
    fi
    
    # 3. 检查测试
    print_step "3. 运行测试"
    if go test ./... -v; then
        print_info "   ✓ 测试通过"
    else
        print_warning "   ⚠ 测试失败或无测试"
    fi
    
    # 4. 检查 Git 状态
    print_step "4. 检查 Git 状态"
    if [ -d "../../.git" ]; then
        cd ../..
        
        if [ -n "$(git status --porcelain rag-sdks/go)" ]; then
            print_warning "   ⚠ 有未提交的更改"
            git status --porcelain rag-sdks/go
        else
            print_info "   ✓ 工作区干净"
        fi
        
        cd - > /dev/null
    else
        print_warning "   ⚠ 不在 Git 仓库中"
    fi
    
    # 5. 检查 go.mod
    print_step "5. 检查 go.mod"
    if grep -q "^go 1\." go.mod; then
        GO_VERSION_IN_MOD=$(grep '^go ' go.mod | awk '{print $2}')
        print_info "   ✓ Go 版本: ${GO_VERSION_IN_MOD}"
    else
        print_warning "   ⚠ go.mod 中没有指定 Go 版本"
    fi
    
    echo ""
    print_info "========================================="
    print_info "✅ 检查完成"
    print_info "========================================="
    echo ""
    print_info "准备发布请参考: ./deploy.sh"
    
else
    # 显示发布指南
    echo ""
    print_info "Go modules 发布指南"
    print_info "--------------------"
    echo ""
    print_info "Go SDK 通过 Git tags 发布，不需要额外的发布工具。"
    echo ""
    
    print_step "发布流程:"
    echo ""
    echo "1. 确保代码已提交到 Git"
    echo "   cd ../../"
    echo "   git add rag-sdks/go"
    echo "   git commit -m \"feat(go-sdk): release version vX.Y.Z\""
    echo ""
    
    echo "2. 创建 Git tag (必须在仓库根目录)"
    echo "   cd ../../"
    echo "   git tag rag-sdks/go/vX.Y.Z"
    echo "   示例: git tag rag-sdks/go/v1.0.0"
    echo ""
    
    echo "3. 推送代码和 tag"
    echo "   git push origin master"
    echo "   git push origin rag-sdks/go/vX.Y.Z"
    echo ""
    
    echo "4. 发布完成！用户可以这样使用:"
    echo "   go get ${MODULE_PATH}@vX.Y.Z"
    echo ""
    
    print_warning "注意事项:"
    echo ""
    echo "• Tag 格式必须是: rag-sdks/go/vX.Y.Z (因为这是子模块)"
    echo "• 版本号必须遵循语义化版本规范 (vX.Y.Z)"
    echo "• 推送到 GitHub 后，Go proxy 会自动索引"
    echo "• 通常几分钟内就可以通过 go get 安装"
    echo ""
    
    print_info "快速发布命令 (需要在仓库根目录执行):"
    echo ""
    echo "VERSION=v1.0.0  # 修改为你的版本号"
    echo "cd ../../  # 回到仓库根目录"
    echo "git add rag-sdks/go"
    echo "git commit -m \"feat(go-sdk): release version \$VERSION\""
    echo "git tag rag-sdks/go/\$VERSION"
    echo "git push origin master"
    echo "git push origin rag-sdks/go/\$VERSION"
    echo ""
    
    print_info "检查发布准备情况:"
    echo "  ./deploy.sh check"
    echo ""
    
    print_info "查看已发布的版本:"
    echo "  git tag -l 'rag-sdks/go/v*'"
    echo ""
fi

