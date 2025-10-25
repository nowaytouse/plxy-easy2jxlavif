# Media Tools - 媒体文件辅助工具集

统一的媒体文件管理工具，集成XMP元数据合并、重复检测和文件规范化功能。

## 功能特性

### 1. XMP元数据合并 (`merge`命令)
- 将XMP侧边文件合并到对应的媒体文件
- 支持 `.xmp` 和 `.sidecar.xmp` 格式
- 自动检测对应的媒体文件
- 使用ExifTool进行可靠的元数据传输
- 支持包含空格和特殊字符的路径

### 2. 重复媒体检测 (`dedup`命令)
- 基于SHA256哈希的重复检测
- 将重复文件移动到垃圾箱
- 支持所有常见媒体格式
- 安全删除并验证
- 支持包含空格和特殊字符的路径

### 3. 文件扩展名规范化 (`normalize`命令)
- 标准化文件扩展名：`.jpeg` → `.jpg`、`.tiff` → `.tif`
- 不区分大小写检测
- 批量处理
- 支持试运行模式

### 4. 自动处理 (`auto`命令) - **推荐使用**
- 按正确顺序执行所有操作：
  1. 规范化扩展名
  2. 合并XMP元数据
  3. 检测并移除重复文件
- 一条命令完成完整的媒体管理

## 安装

```bash
./build.sh
```

## 使用方法

### 自动处理（推荐）

```bash
# 一条命令完成完整的媒体管理
./bin/media_tools auto -dir /path/to/media -trash /path/to/trash

# 试运行模式（仅预览）
./bin/media_tools auto -dir /path/to/media -trash /path/to/trash -dry-run

# 支持包含空格的路径
./bin/media_tools auto -dir "/path/with spaces/media" -trash "/path/to/trash"
```

### 单独操作

#### 规范化文件扩展名
```bash
# 标准化扩展名 (.jpeg→.jpg, .tiff→.tif)
./bin/media_tools normalize -dir /path/to/media

# 预览更改
./bin/media_tools normalize -dir /path/to/media -dry-run
```

#### 合并XMP元数据
```bash
# 合并XMP侧边文件
./bin/media_tools merge -dir /path/to/media

# 预览合并
./bin/media_tools merge -dir /path/to/media -dry-run
```

#### 去重媒体文件
```bash
# 移除重复文件
./bin/media_tools dedup -dir /path/to/media -trash /path/to/trash

# 预览重复文件
./bin/media_tools dedup -dir /path/to/media -trash /path/to/trash -dry-run
```

## 系统要求

- Go 1.25+
- ExifTool（用于元数据操作）

## 使用示例

```bash
# 完整自动处理（推荐）
./bin/media_tools auto -dir ~/Pictures/PhotoLibrary -trash ~/Pictures/.trash

# 支持中文字符和空格
./bin/media_tools auto -dir "~/图片/照片 (2024)" -trash "~/图片/.trash"

# 单独操作
./bin/media_tools normalize -dir ~/Pictures  # 步骤1
./bin/media_tools merge -dir ~/Pictures      # 步骤2
./bin/media_tools dedup -dir ~/Pictures -trash ~/Pictures/.trash  # 步骤3

# 预览所有操作，不实际执行
./bin/media_tools auto -dir ~/Pictures -trash ~/Pictures/.trash -dry-run
```

## 版本

2.2.0

## 作者

AI Assistant

