# Archive 归档说明

本目录包含EasyMode项目的归档文件，按类型分类存储。

## 📁 目录结构

### duplicates/ - 重复文件
- `.gitignore 2` - 重复的gitignore文件

### logs/ - 日志文件
- `media_tools.log` - 媒体工具日志
- `universal_converter.log` - 通用转换器日志

### old_docs/ - 历史文档
- `v2.1.0/` - v2.1.0版本相关文档
  - `TEST_REPORT_v2.1.0.md` - 测试报告
  - `UPDATE_SUMMARY_v2.1.0.md` - 更新摘要
- `v2.1.1/` - v2.1.1版本相关文档
  - `ENHANCEMENT_REPORT_v2.1.1.md` - 增强报告
  - `FINAL_SUMMARY_v2.1.1.md` - 最终总结
- `test_reports/` - 测试报告
  - `COMPREHENSIVE_TEST_REPORT.md` - 综合测试报告
- 其他历史文档
  - `FINAL_COMPREHENSIVE_REPORT_v2.2.0.md` - v2.2.0最终报告
  - `OPTIMIZATION_v2.2.1.md` - 优化报告
  - `USAGE_TUTORIAL.md` - 旧版使用教程
  - `USAGE_TUTORIAL_ZH.md` - 旧版中文教程

### old_versions/ - 旧版本代码
- `Single version/` - 单一功能版本工具
  - `all2avif/` - AVIF转换工具
  - `all2jxl/` - JXL转换工具
  - `dynamic2avif/` - 动态图像转AVIF
  - `dynamic2jxl/` - 动态图像转JXL
  - `static2avif/` - 静态图像转AVIF
  - `static2jxl/` - 静态图像转JXL
  - `video2mov/` - 视频转MOV
  - `merge_xmp/` - XMP合并工具
  - `deduplicate_media/` - 媒体去重工具

### scripts/ - 脚本文件
- `update_filetype_detection.sh` - 文件类型检测更新脚本

### junk/ - 垃圾文件
- `ds_store/` - 所有.DS_Store文件
- `logs/` - 历史日志文件
- `duplicates/` - 重复文件
- `build_all.sh` - 旧版构建脚本

### trash/ - 临时文件
- 各种临时和垃圾文件

### final_trash/ - 最终垃圾文件
- 处理过程中的临时文件

## 📋 整理说明

### 整理时间
- **整理日期**: 2025-10-24
- **整理原因**: 清理冗余文件，优化项目结构

### 整理原则
1. **保留历史**: 所有历史版本和文档都保留在archive目录
2. **分类存储**: 按文件类型和版本分类存储
3. **便于查找**: 清晰的目录结构和说明文档
4. **避免重复**: 移除重复文件，保留最新版本

### 当前活跃目录
- `docs/` - 当前版本文档
- `universal_converter/` - 通用转换器
- `media_tools/` - 媒体工具集
- `utils/` - 工具库
- `test_data/` - 测试数据

## 🔍 查找文件

### 查找历史文档
```bash
# 查找v2.1.0相关文档
ls archive/old_docs/v2.1.0/

# 查找测试报告
ls archive/old_docs/test_reports/

# 查找旧版本工具
ls archive/old_versions/Single\ version/
```

### 查找日志文件
```bash
# 查找所有日志
find archive/logs/ -name "*.log"
```

### 查找脚本文件
```bash
# 查找所有脚本
find archive/scripts/ -name "*.sh"
```

## ⚠️ 注意事项

1. **不要删除**: archive目录中的文件都是历史记录，请勿删除
2. **定期清理**: 可以定期清理trash目录中的临时文件
3. **版本管理**: 新版本文件不要放入archive，应放在对应目录
4. **文档更新**: 更新文档时请同时更新本说明文件

---

**归档版本**: v2.2.0  
**整理日期**: 2025-10-24  
**维护者**: AI Assistant
