package i18n

func getEnUSMessages() map[string]string {
	return map[string]string{
		// General messages
		MsgWelcome:          "Welcome to Pixly Image Converter",
		MsgStarting:         "Starting...",
		MsgCompleted:        "Completed!",
		MsgFailed:           "Failed",
		MsgProgress:         "Progress: {0}",
		MsgSuccess:          "Success",
		MsgError:            "Error",
		MsgWarning:          "Warning",
		
		// File operations
		MsgScanningFiles:    "Scanning files...",
		MsgProcessingFile:   "Processing: {0}",
		MsgFileCompleted:    "✅ File completed: {0}",
		MsgFileFailed:       "❌ File failed: {0}",
		MsgFileSkipped:      "⏭️  File skipped: {0}",
		
		// Conversion
		MsgConversionStart:  "Starting conversion: {0} → {1}",
		MsgConversionDone:   "Conversion done: {0}",
		MsgSpaceSaved:       "Space saved: {0}% ({1} → {2})",
		MsgQualityAnalysis:  "Quality analysis: {0}",
		
		// Session
		MsgSessionCreate:    "Creating session: {0}",
		MsgSessionLoad:      "Loading session: {0}",
		MsgSessionComplete:  "Session completed: {0}",
		MsgSessionResume:    "Resuming session: {0}",
		
		// Error messages
		ErrFileNotFound:     "File not found: {0}",
		ErrPermissionDenied: "Permission denied: {0}",
		ErrDiskFull:         "Disk space full",
		ErrInvalidFormat:    "Invalid format: {0}",
		ErrConversionFailed: "Conversion failed: {0}",
		
		// Statistics
		StatTotalFiles:      "Total files: {0}",
		StatProcessed:       "Processed: {0}",
		StatSucceeded:       "Succeeded: {0}",
		StatFailed:          "Failed: {0}",
		StatSkipped:         "Skipped: {0}",
		StatSpaceSaved:      "Space saved: {0}",
		StatDuration:        "Duration: {0}",
		
		// CLI messages
		"cli.help":               "Show help information",
		"cli.version":            "Show version information",
		"cli.input":              "Input directory or file",
		"cli.output":             "Output directory",
		"cli.format":             "Target format",
		"cli.quality":            "Quality settings",
		"cli.workers":            "Number of concurrent workers",
		"cli.recursive":          "Process subdirectories recursively",
		"cli.overwrite":          "Overwrite existing files",
		"cli.in_place":           "In-place conversion (delete originals)",
		"cli.resume":             "Resume previous session",
		"cli.list_sessions":      "List all sessions",
		"cli.lang":               "Interface language",
		
		// Performance monitoring
		"perf.cpu_usage":         "CPU usage: {0}%",
		"perf.memory_usage":      "Memory: {0}MB",
		"perf.disk_io":           "Disk I/O: {0}MB/s",
		"perf.worker_count":      "Workers: {0}",
		"perf.worker_adjusting":  "Adjusting workers: {0} → {1}",
		
		// Quality analysis
		"quality.analyzing":      "Analyzing quality...",
		"quality.level.extreme":  "Extreme",
		"quality.level.high":     "High",
		"quality.level.medium":   "Medium",
		"quality.level.low":      "Low",
		"quality.level.very_low": "Very Low",
		"quality.content.photo":  "Photo",
		"quality.content.graphic":"Graphic",
		"quality.content.screenshot":"Screenshot",
		"quality.content.mixed":  "Mixed",
		
		// Checkpoint
		"checkpoint.saving":      "Saving progress...",
		"checkpoint.saved":       "Progress saved",
		"checkpoint.loading":     "Loading progress...",
		"checkpoint.loaded":      "Progress loaded",
		"checkpoint.found":       "Found incomplete session",
		"checkpoint.resume_ask":  "Resume session? (y/n)",
		
		// Configuration
		"config.loading":         "Loading configuration...",
		"config.loaded":          "Configuration loaded",
		"config.invalid":         "Invalid configuration: {0}",
		"config.not_found":       "Configuration file not found",
		
		// File types
		"format.jpg":             "JPEG Image",
		"format.png":             "PNG Image",
		"format.gif":             "GIF Animation",
		"format.webp":            "WebP Image",
		"format.jxl":             "JPEG XL Image",
		"format.avif":            "AVIF Image",
		"format.heic":            "HEIC Image",
		
		// Hints
		"hint.drag_drop":         "Hint: Drag and drop files or folders",
		"hint.ctrl_c":            "Hint: Press Ctrl+C to exit anytime",
		"hint.resume_available":  "Hint: Use --resume to continue last session",
		"hint.check_log":         "Hint: Check logs for details",
		
		// Confirmation
		"confirm.continue":       "Continue?",
		"confirm.delete_original":"Delete original files?",
		"confirm.overwrite":      "File exists, overwrite?",
		"confirm.yes":            "Yes",
		"confirm.no":             "No",
		
		// Report
		"report.generating":      "Generating report...",
		"report.saved":           "Report saved: {0}",
		"report.title":           "Conversion Report",
		"report.summary":         "Summary",
		"report.details":         "Details",
	}
}
