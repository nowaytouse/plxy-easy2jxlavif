# 更新日志 (Changelog)

## [2025-10-22] - 安全删除与统计信息修复版本

### 新增功能
- **安全删除机制**: 实施了严格的安全删除检查，确保只有在确认目标文件存在且有效的前提下才删除原始文件
  - 为具有 `-replace` 参数的工具添加了安全删除验证
  - 只有明确带有 `-replace` 参数的工具才会删除原始文件
  - 其他工具（如 `static2avif`, `dynamic2avif`, `static2jxl`, `dynamic2jxl` 等）默认不会删除原始文件，只会生成转换后的文件

### 修复内容
- **HEIC/HEIF文件转换失败问题**: 修复了all2jxl、static2jxl、all2avif、static2avif等工具无法正确转换HEIC文件的问题
- **增强的HEIC处理逻辑**: 添加了多种备选方案处理复杂HEIC文件
  - 使用ImageMagick的增强选项（`-define heic:limit-num-tiles=0`, `-define heic:max-image-size=0`, `-define heic:use-embedded-profile=false`）
  - 集成FFmpeg作为备选转换方案
  - 自动检测HEIC文件尺寸并进行适当缩放
  - 添加更多放松的ImageMagick策略
- **统计信息准确性**: 修复了节省空间计算的错误
  - 当转换后的文件比原始文件更大时不显示负数节省空间
  - 压缩率现在正确显示转换效率（>100% 表示文件变大）

### 具体修改内容
1. **all2jxl**: 
   - 在HEIC转换流程中添加了三种转换方法：ImageMagick增强模式、FFmpeg精确转换、ImageMagick放松模式
   - 改进了尺寸检测逻辑以支持高分辨率HEIC文件
   - 优化了错误处理与日志记录
   - 添加了安全删除验证机制

2. **static2jxl**: 
   - 相同的HEIC转换增强功能
   - 正确处理了依赖关系和模块配置
   - 添加了安全删除验证机制

3. **all2avif**: 
   - 增强了HEIC转换逻辑，添加了多重备选方案
   - 引入了改进的ImageMagick和FFmpeg处理流程
   - 优化了错误处理和日志记录
   - 添加了安全删除验证机制

4. **static2avif**: 
   - 增强了HEIC转换逻辑，添加了多重备选方案
   - 与all2avif采用相同的标准转换流程
   - 添加了安全删除验证机制

5. **其他相关脚本**:
   - 检查并确保所有处理HEIC文件的脚本都使用了相同的增强转换逻辑
   - 为 video2mov 脚本添加了安全删除验证机制
   - 为 merge_xmp 脚本添加了安全删除验证机制

### 技术改进
- 处理安全限制超限问题（如"Maximum number of child boxes in 'ipco' box exceeded"）
- 改进内存管理，避免大尺寸HEIC文件转换时的内存分配错误
- 优化转换流程以支持更多复杂的HEIC文件格式
- 统一了所有工具的HEIC处理策略，提高一致性和可靠性
- 实现了安全删除机制，防止意外删除原始文件

### 测试结果
- 所有测试的HEIC文件现在都能成功转换为JXL和AVIF格式
- 保持了原有的元数据迁移功能
- 性能影响最小化
- 转换成功率显著提高
- 安全删除机制经过测试验证，确保原始文件只在目标文件存在且有效时才被删除