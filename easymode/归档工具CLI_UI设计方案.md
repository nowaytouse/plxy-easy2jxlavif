# 🎨 归档工具统一CLI UI设计方案

**设计目标**: 为所有归档工具提供简易、安全、美观的CLI交互界面  
**核心理念**: 拖入目录 → 回车 → 自动开始

---

## 🎯 设计需求

### 用户需求

1. ✅ **极简交互**: 拖入目录 → 回车 → 一键开始
2. ✅ **强大安全**: 参考Pixly的安全检查机制
3. ✅ **双模式支持**: 
   - 交互模式（拖拽式）
   - 非交互模式（命令行参数）
4. ✅ **元数据保留**: 自动保留所有元数据（内部+文件系统+Finder）

### 安全需求（参考Pixly）

1. ✅ **路径安全检查**: 
   - 禁止系统目录（/System, /usr/bin等）
   - 警告敏感目录（/Applications, 用户根目录）
   - 允许安全目录（Documents, Desktop等）

2. ✅ **权限预检**: 
   - 验证读写权限
   - 检查磁盘空间
   - 验证路径存在性

3. ✅ **用户确认**: 
   - 敏感目录需要二次确认
   - 显示完整路径信息
   - 提供安全建议

---

## 🏗️ 架构设计

### 统一框架模块

**文件**: `easymode/utils/cli_ui.go`

**功能模块**:

```go
// 1. 交互模式管理
type InteractiveMode struct {
    Config *CLIConfig
}

// 2. 核心功能
func ShowBanner()                    // 显示横幅
func PromptForDirectory() string     // 提示输入目录
func PerformSafetyCheck(path) error  // 安全检查
func ShowProgress(current, total)    // 显示进度
func ShowSummary(stats)              // 显示总结

// 3. 安全功能
func isCriticalSystemPath(path) bool  // 系统目录检查
func isSensitiveDirectory(path) bool  // 敏感目录检查
func getDiskSpace(path) uint64        // 磁盘空间
func unescapeShellPath(path) string   // macOS路径转义
```

---

## 🎨 UI设计

### 启动界面

```
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   🎨 static2jxl v2.3.0                                        ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

📁 请拖入要处理的文件夹，然后按回车键：
   （或直接输入路径）

路径: _
```

### 安全检查界面

```
🔍 正在执行安全检查...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✅ 路径存在: /Users/nyamiiko/Documents/test
  ✅ 路径安全: 非系统目录
  ✅ 权限验证: 可读可写
  💾 磁盘空间: 245.3GB / 500GB (49.1% 可用)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ 安全检查通过！

📊 扫描到 128 个支持的文件
🔄 开始处理...
```

### 处理进度界面

```
🎨 进度: [45/128] 35.2% - 正在处理: IMG_1234.jpg

✅ IMG_1234.jpg → IMG_1234.jxl
   文件内部元数据: ✅ EXIF/XMP/GPS
   文件系统元数据: ✅ 2024-01-15 10:30:00
   Finder扩展属性: ✅ 标签/注释
   空间节省: 2.5MB → 1.2MB (52% ↓)
```

### 完成界面

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 处理完成
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  总计: 128 个文件
  ✅ 成功: 125 个
  ❌ 失败: 3 个
  📈 成功率: 97.7%
  💾 空间节省: 256MB → 145MB (43.4% ↓)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 🔧 实现方案

### 每个归档工具的main.go修改

**修改前** (命令行模式):
```go
func main() {
    flag.Parse()
    
    if *inputDir == "" {
        log.Fatal("请使用 -dir 参数指定目录")
    }
    
    // ... 直接开始处理 ...
}
```

**修改后** (双模式):
```go
func main() {
    flag.Parse()
    
    var targetDir string
    
    // 检测模式
    if len(os.Args) == 1 {
        // 🎨 交互模式：无参数启动
        runInteractiveMode()
    } else if *inputDir != "" {
        // 📝 非交互模式：命令行参数
        targetDir = *inputDir
        processDirectory(targetDir, opts)
    } else {
        showHelp()
    }
}

func runInteractiveMode() {
    // 1. 显示横幅
    ui := utils.NewInteractiveMode(&utils.CLIConfig{
        ToolName: "static2jxl",
        ToolVersion: "v2.3.0",
        SupportedExts: []string{".jpg", ".png", ".gif"},
        OutputFormat: "JXL",
    })
    
    ui.ShowBanner()
    
    // 2. 提示输入目录
    targetDir, err := ui.PromptForDirectory()
    if err != nil {
        log.Fatalf("❌ %v", err)
    }
    
    // 3. 安全检查
    if err := ui.PerformSafetyCheck(targetDir); err != nil {
        log.Fatalf("❌ 安全检查失败: %v", err)
    }
    
    // 4. 开始处理
    processDirectory(targetDir, opts)
}
```

---

## 🛡️ 安全检查策略

### 三级安全分类

#### 1. 系统级禁止路径（Critical）- 绝对拒绝

**macOS**:
```
/System
/Library/System
/private
/usr/bin
/usr/sbin
/bin
/sbin
/var/root
/etc
/dev
/proc
/Applications/Utilities
```

**行为**: 
- ❌ 直接拒绝
- 显示错误信息
- 提供安全目录建议

---

#### 2. 警告级路径（Warning）- 需要确认

**macOS**:
```
/Applications
/Library
/usr
/var
~/（用户根目录）
```

**行为**:
- ⚠️ 显示警告
- 要求用户输入"yes"确认
- 提供风险说明

---

#### 3. 安全路径（Safe）- 直接允许

**macOS**:
```
~/Documents
~/Desktop
~/Downloads
~/Pictures
~/Movies
~/Music
/tmp
/Users/Shared
*_test（测试目录）
*/测试*（中文测试目录）
```

**行为**:
- ✅ 直接允许
- 快速通过安全检查
- 无需用户确认

---

## 📝 使用示例

### 交互模式（推荐）

```bash
cd /Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive/static2jxl
./bin/static2jxl-darwin-arm64

# 输出:
# ╔════════════════════════════════════════════════════════════╗
# ║                                                            ║
# ║   🎨 static2jxl v2.3.0                                     ║
# ║                                                            ║
# ╚════════════════════════════════════════════════════════════╝
# 
# 📁 请拖入要处理的文件夹，然后按回车键：
#    （或直接输入路径）
# 
# 路径: <拖入文件夹>
# 
# 🔍 正在执行安全检查...
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#   ✅ 路径存在: /Users/nyamiiko/Documents/test
#   ✅ 路径安全: 非系统目录
#   ✅ 权限验证: 可读可写
#   💾 磁盘空间: 245.3GB / 500GB (49.1% 可用)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# ✅ 安全检查通过！
# 
# 🔄 开始处理...
```

### 非交互模式

```bash
./bin/static2jxl-darwin-arm64 -dir /path/to/folder -workers 4

# 输出:
# ✅ 扫描到 45 个文件
# 🔄 开始处理...
# ✅ IMG_1234.jpg → IMG_1234.jxl
# ...
```

---

## 🚀 实现计划

### 阶段一: 创建统一框架 ✅

- [x] 创建 `easymode/utils/cli_ui.go`
- [x] 实现交互模式框架
- [x] 实现安全检查功能
- [x] 实现进度显示

### 阶段二: 集成到归档工具

**需要修改的工具** (5个):
1. ⏳ dynamic2avif
2. ⏳ video2mov
3. ⏳ static2jxl
4. ⏳ static2avif
5. ⏳ dynamic2jxl

**修改内容**:
- 添加交互模式入口
- 集成安全检查
- 保留命令行模式
- 统一UI风格

### 阶段三: 测试验证

- ⏳ 测试交互模式
- ⏳ 测试非交互模式
- ⏳ 测试安全检查
- ⏳ 验证元数据保留

---

## 📊 对比表

| 特性 | 修改前 | 修改后 |
|------|--------|--------|
| 启动方式 | 仅命令行参数 | 双模式（交互+命令行） |
| 路径输入 | -dir参数 | 拖拽+回车 |
| 安全检查 | 无 | 强大安全检查 |
| 用户体验 | 命令行专业用户 | 普通用户也可用 |
| 错误提示 | 简单 | 详细+建议 |
| 进度显示 | 日志 | 美观进度条 |
| 元数据保留 | 部分 | 100%完整 |

---

## 🎊 预期效果

### 普通用户使用流程

```
1. 双击启动工具
   ↓
2. 拖入文件夹
   ↓
3. 按回车键
   ↓
4. 自动安全检查
   ↓
5. 开始转换（带进度）
   ↓
6. 显示完成总结
```

**时间**: <30秒（包括安全检查）  
**操作**: 2步（拖入+回车）  
**专业度**: ⭐⭐⭐（强大安全+完美元数据保留）

---

## 📋 下一步行动

1. ✅ 创建统一CLI UI框架（utils/cli_ui.go）
2. ⏳ 为5个归档工具添加交互模式
3. ⏳ 集成安全检查
4. ⏳ 测试所有工具
5. ⏳ 创建使用说明文档

---

**开始实现？**

