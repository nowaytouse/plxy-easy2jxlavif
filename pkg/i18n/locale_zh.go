package i18n

func getZhCNMessages() map[string]string {
	return map[string]string{
		// 通用消息
		MsgWelcome:          "欢迎使用Pixly图像转换工具",
		MsgStarting:         "正在启动...",
		MsgCompleted:        "完成！",
		MsgFailed:           "失败",
		MsgProgress:         "进度: {0}",
		MsgSuccess:          "成功",
		MsgError:            "错误",
		MsgWarning:          "警告",
		
		// 文件操作
		MsgScanningFiles:    "正在扫描文件...",
		MsgProcessingFile:   "正在处理: {0}",
		MsgFileCompleted:    "✅ 文件完成: {0}",
		MsgFileFailed:       "❌ 文件失败: {0}",
		MsgFileSkipped:      "⏭️  文件跳过: {0}",
		
		// 转换
		MsgConversionStart:  "开始转换: {0} → {1}",
		MsgConversionDone:   "转换完成: {0}",
		MsgSpaceSaved:       "节省空间: {0}% ({1} → {2})",
		MsgQualityAnalysis:  "质量分析: {0}",
		
		// 会话
		MsgSessionCreate:    "创建会话: {0}",
		MsgSessionLoad:      "加载会话: {0}",
		MsgSessionComplete:  "会话完成: {0}",
		MsgSessionResume:    "恢复会话: {0}",
		
		// 错误消息
		ErrFileNotFound:     "文件不存在: {0}",
		ErrPermissionDenied: "权限不足: {0}",
		ErrDiskFull:         "磁盘空间不足",
		ErrInvalidFormat:    "格式无效: {0}",
		ErrConversionFailed: "转换失败: {0}",
		
		// 统计
		StatTotalFiles:      "总文件数: {0}",
		StatProcessed:       "已处理: {0}",
		StatSucceeded:       "成功: {0}",
		StatFailed:          "失败: {0}",
		StatSkipped:         "跳过: {0}",
		StatSpaceSaved:      "空间节省: {0}",
		StatDuration:        "耗时: {0}",
		
		// CLI消息
		"cli.help":               "显示帮助信息",
		"cli.version":            "显示版本信息",
		"cli.input":              "输入目录或文件",
		"cli.output":             "输出目录",
		"cli.format":             "目标格式",
		"cli.quality":            "质量设置",
		"cli.workers":            "并发工作线程数",
		"cli.recursive":          "递归处理子目录",
		"cli.overwrite":          "覆盖已存在文件",
		"cli.in_place":           "原地转换（删除原文件）",
		"cli.resume":             "恢复上次会话",
		"cli.list_sessions":      "列出所有会话",
		"cli.lang":               "界面语言",
		
		// 性能监控
		"perf.cpu_usage":         "CPU使用率: {0}%",
		"perf.memory_usage":      "内存使用: {0}MB",
		"perf.disk_io":           "磁盘IO: {0}MB/s",
		"perf.worker_count":      "工作线程: {0}",
		"perf.worker_adjusting":  "调整工作线程: {0} → {1}",
		
		// 质量分析
		"quality.analyzing":      "分析质量...",
		"quality.level.extreme":  "极高",
		"quality.level.high":     "高",
		"quality.level.medium":   "中",
		"quality.level.low":      "低",
		"quality.level.very_low": "极低",
		"quality.content.photo":  "照片",
		"quality.content.graphic":"图形",
		"quality.content.screenshot":"截图",
		"quality.content.mixed":  "混合",
		
		// 断点续传
		"checkpoint.saving":      "保存进度...",
		"checkpoint.saved":       "进度已保存",
		"checkpoint.loading":     "加载进度...",
		"checkpoint.loaded":      "进度已加载",
		"checkpoint.found":       "发现未完成的会话",
		"checkpoint.resume_ask":  "是否恢复会话? (y/n)",
		
		// 配置
		"config.loading":         "加载配置...",
		"config.loaded":          "配置已加载",
		"config.invalid":         "配置无效: {0}",
		"config.not_found":       "配置文件不存在",
		
		// 文件类型
		"format.jpg":             "JPEG图像",
		"format.png":             "PNG图像",
		"format.gif":             "GIF动图",
		"format.webp":            "WebP图像",
		"format.jxl":             "JPEG XL图像",
		"format.avif":            "AVIF图像",
		"format.heic":            "HEIC图像",
		
		// 提示
		"hint.drag_drop":         "提示: 可直接拖入文件或目录",
		"hint.ctrl_c":            "提示: 按Ctrl+C随时退出",
		"hint.resume_available":  "提示: 使用 --resume 恢复上次会话",
		"hint.check_log":         "提示: 查看日志了解详情",
		
		// 确认
		"confirm.continue":       "是否继续?",
		"confirm.delete_original":"是否删除原文件?",
		"confirm.overwrite":      "文件已存在，是否覆盖?",
		"confirm.yes":            "是",
		"confirm.no":             "否",
		
		// 报告
		"report.generating":      "生成报告...",
		"report.saved":           "报告已保存: {0}",
		"report.title":           "转换报告",
		"report.summary":         "总结",
		"report.details":         "详细信息",
	}
}
