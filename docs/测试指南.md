# Pixly 测试套件使用指南

## 🧪 测试框架概述

Pixly 配备了完整的统一测试框架，支持多种测试类型，确保代码质量和功能稳定性。测试框架采用模块化设计，支持自动化执行和详细报告生成。

## 📋 测试套件组成

### 核心组件
```
test/
├── unified_test_manager.go      # 统一测试管理器
├── unified_test_executor.go     # 测试执行器
├── unified_test_config.json     # 测试配置文件
├── ui_interaction_test.go       # UI交互测试
├── headless_converter_test.go   # 无头转换器测试
├── ai_test_tool.go             # AI测试工具
└── test_data/                  # 测试数据目录
    ├── images/                 # 测试图片
    ├── videos/                 # 测试视频
    └── expected/               # 期望结果
```

### 测试类型
1. **UI交互测试**: 验证用户界面功能
2. **转换器测试**: 验证核心转换逻辑
3. **依赖检查测试**: 验证系统依赖
4. **性能基准测试**: 验证处理性能
5. **集成测试**: 验证模块间交互
6. **回归测试**: 验证功能稳定性

## 🚀 快速开始

### 1. 环境准备
```bash
# 确保Go环境正确配置
go version

# 安装测试依赖
go mod tidy

# 验证测试环境
go test -v ./...
```

### 2. 运行完整测试套件
```bash
# 编译测试执行器
go build -o unified_test_executor test/unified_test_executor.go

# 运行完整测试套件
./unified_test_executor

# 或者使用Go直接运行
go run test/unified_test_executor.go
```

### 3. 运行特定测试
```bash
# 运行UI交互测试
go test -v ./test -run TestUIInteraction

# 运行转换器测试
go test -v ./test -run TestHeadlessConverter

# 运行依赖检查测试
go test -v ./test -run TestDependencyCheck
```

## ⚙️ 测试配置

### 配置文件结构 (`unified_test_config.json`)
```json
{
  "test_settings": {
    "timeout_seconds": 300,
    "max_concurrent_tests": 4,
    "enable_performance_tests": true,
    "enable_ui_tests": true,
    "enable_integration_tests": true
  },
  "test_data": {
    "input_directory": "test/test_data/images",
    "output_directory": "test/test_output",
    "expected_directory": "test/test_data/expected"
  },
  "conversion_tests": {
    "test_modes": ["auto+", "quality", "emoji"],
    "test_formats": ["jpg", "png", "gif", "webp"],
    "quality_thresholds": {
      "min_compression_ratio": 0.05,
      "max_quality_loss": 0.1
    }
  },
  "performance_tests": {
    "benchmark_files": [
      "test_small.jpg",
      "test_medium.png",
      "test_large.gif"
    ],
    "performance_thresholds": {
      "max_processing_time_ms": 5000,
      "max_memory_usage_mb": 512
    }
  }
}
```

### 自定义配置
```bash
# 使用自定义配置文件
./unified_test_executor --config custom_test_config.json

# 设置特定参数
./unified_test_executor --timeout 600 --concurrent 8
```

## 🔧 详细测试说明

### 1. UI交互测试 (UI Interaction Tests)

#### 测试内容
- 菜单导航功能
- 用户输入处理
- 进度条显示
- 错误提示界面
- 配置界面交互

#### 运行方式
```bash
# 单独运行UI测试
go test -v ./test -run TestUIInteraction

# 带详细输出
go test -v ./test -run TestUIInteraction -args -verbose

# 测试特定UI组件
go test -v ./test -run TestUIInteraction/MenuNavigation
```

#### 测试示例
```go
func TestUIMenuNavigation(t *testing.T) {
    // 模拟用户输入
    input := []string{"1", "2", "q"}
    
    // 创建测试环境
    testUI := NewTestUI(input)
    
    // 执行测试
    result := testUI.RunMenuTest()
    
    // 验证结果
    assert.True(t, result.Success)
    assert.Contains(t, result.Output, "主菜单")
}
```

### 2. 转换器测试 (Converter Tests)

#### 测试内容
- 图片格式转换
- 视频格式转换
- 质量参数验证
- 错误处理机制
- 性能基准测试

#### 运行方式
```bash
# 运行所有转换器测试
go test -v ./test -run TestHeadlessConverter

# 测试特定格式转换
go test -v ./test -run TestHeadlessConverter/JPEG_to_AVIF

# 性能基准测试
go test -v ./test -bench=BenchmarkConversion
```

#### 测试示例
```go
func TestJPEGToAVIFConversion(t *testing.T) {
    // 准备测试文件
    inputFile := "test_data/images/test.jpg"
    
    // 创建转换器
    converter := NewHeadlessConverter()
    
    // 执行转换
    result, err := converter.ConvertToAVIF(inputFile, 75)
    
    // 验证结果
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Greater(t, result.CompressionRatio, 0.05)
}
```

### 3. 依赖检查测试 (Dependency Tests)

#### 测试内容
- 外部工具可用性
- 版本兼容性检查
- 安装状态验证
- 功能完整性测试

#### 运行方式
```bash
# 运行依赖检查测试
go test -v ./test -run TestDependencyCheck

# 检查特定工具
go test -v ./test -run TestDependencyCheck/FFmpeg
```

### 4. 性能基准测试 (Performance Tests)

#### 测试内容
- 转换速度基准
- 内存使用监控
- 并发性能测试
- 大文件处理测试

#### 运行方式
```bash
# 运行性能基准测试
go test -v ./test -bench=.

# 运行特定基准测试
go test -v ./test -bench=BenchmarkLargeFileConversion

# 生成性能报告
go test -v ./test -bench=. -benchmem > performance_report.txt
```

## 📊 测试报告

### 报告生成
```bash
# 生成详细测试报告
./unified_test_executor --report detailed

# 生成HTML报告
./unified_test_executor --report html --output test_report.html

# 生成JSON报告
./unified_test_executor --report json --output test_report.json
```

### 报告内容
- **测试概要**: 总体通过率和执行时间
- **详细结果**: 每个测试的具体结果
- **性能指标**: 转换速度和资源使用
- **覆盖率报告**: 代码覆盖率统计
- **错误分析**: 失败测试的详细分析

### 报告示例
```
=== Pixly 测试套件报告 ===
执行时间: 2025-09-02 19:45:23
总测试数: 156
通过: 154
失败: 2
跳过: 0
通过率: 98.7%

=== 性能指标 ===
JPEG转AVIF平均时间: 1.2s
内存峰值使用: 245MB
并发效率: 85%

=== 失败测试 ===
1. TestLargeVideoConversion: 超时
2. TestCorruptedFileHandling: 断言失败
```

## 🔍 调试和故障排除

### 调试模式
```bash
# 启用调试模式
./unified_test_executor --debug

# 详细日志输出
./unified_test_executor --verbose --log-level debug

# 保留测试文件
./unified_test_executor --keep-temp-files
```

### 常见问题

#### 1. 测试超时
```bash
# 增加超时时间
./unified_test_executor --timeout 600

# 减少并发数
./unified_test_executor --concurrent 2
```

#### 2. 依赖缺失
```bash
# 检查依赖状态
./pixly deps

# 跳过依赖相关测试
./unified_test_executor --skip-dependency-tests
```

#### 3. 权限问题
```bash
# 检查文件权限
ls -la test/test_data/

# 修复权限
chmod -R 755 test/test_data/
```

#### 4. 内存不足
```bash
# 监控内存使用
top -p $(pgrep unified_test_executor)

# 减少测试并发数
./unified_test_executor --concurrent 1
```

## 🧩 自定义测试

### 添加新测试

#### 1. 创建测试文件
```go
// test/custom_test.go
package test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCustomFeature(t *testing.T) {
    // 测试逻辑
    result := CustomFunction()
    assert.True(t, result)
}
```

#### 2. 注册到测试套件
```go
// 在unified_test_manager.go中添加
func (utm *UnifiedTestManager) RegisterCustomTest() {
    utm.tests = append(utm.tests, TestInfo{
        Name:        "CustomFeature",
        Description: "测试自定义功能",
        Function:    TestCustomFeature,
        Category:    "custom",
    })
}
```

### 测试数据管理

#### 添加测试文件
```bash
# 创建测试数据目录
mkdir -p test/test_data/custom

# 添加测试文件
cp sample.jpg test/test_data/custom/
```

#### 配置测试数据
```json
{
  "custom_tests": {
    "test_files": [
      "test/test_data/custom/sample.jpg"
    ],
    "expected_results": {
      "sample.jpg": {
        "target_format": "avif",
        "min_compression": 0.1
      }
    }
  }
}
```

## 📈 持续集成

### CI/CD集成

#### GitHub Actions示例
```yaml
name: Pixly Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - uses: actions/setup-go@v2
      with:
        go-version: 1.21
    
    - name: Install dependencies
      run: |
        sudo apt-get update
        sudo apt-get install ffmpeg imagemagick
    
    - name: Run tests
      run: |
        go mod tidy
        ./unified_test_executor --ci-mode
    
    - name: Upload test reports
      uses: actions/upload-artifact@v2
      with:
        name: test-reports
        path: test_reports/
```

### 质量门禁
```bash
# 设置质量标准
./unified_test_executor --min-coverage 80 --max-failures 0

# CI模式（严格模式）
./unified_test_executor --ci-mode --strict
```

## 🔧 高级功能

### 并行测试
```bash
# 启用并行测试
./unified_test_executor --parallel --workers 8

# 分布式测试（多机器）
./unified_test_executor --distributed --nodes node1,node2,node3
```

### 测试覆盖率
```bash
# 生成覆盖率报告
go test -v ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 集成到测试套件
./unified_test_executor --coverage --coverage-format html
```

### 性能分析
```bash
# CPU性能分析
go test -v ./test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# 内存分析
go test -v ./test -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

## 📚 最佳实践

### 1. 测试编写原则
- **独立性**: 每个测试应该独立运行
- **可重复**: 测试结果应该可重复
- **快速**: 单元测试应该快速执行
- **清晰**: 测试意图应该清晰明确

### 2. 测试数据管理
- **版本控制**: 测试数据纳入版本控制
- **数据隔离**: 不同测试使用独立数据
- **清理机制**: 测试后自动清理临时文件
- **数据更新**: 定期更新测试数据

### 3. 性能测试
- **基准建立**: 建立性能基准线
- **回归检测**: 监控性能回归
- **环境一致**: 保持测试环境一致
- **多次运行**: 多次运行取平均值

### 4. 错误处理
- **异常捕获**: 捕获所有可能的异常
- **错误分类**: 对错误进行分类处理
- **恢复机制**: 提供错误恢复机制
- **日志记录**: 详细记录错误信息

---

**提示**: 测试是保证代码质量的重要手段，建议在开发过程中持续运行测试，确保功能的稳定性和可靠性。