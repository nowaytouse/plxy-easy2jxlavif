package config

// DefaultConfig returns the default Pixly configuration
func DefaultConfig() *Config {
	return &Config{
		Project: ProjectConfig{
			Name:    "Pixly",
			Version: "4.0.0",
			Author:  "Pixly Team",
		},

		Concurrency: ConcurrencyConfig{
			AutoAdjust:        true,
			ConversionWorkers: 8,
			ScanWorkers:       4,
			MemoryLimitMB:     8192,
			EnableMonitoring:  true,
		},

		Conversion: ConversionConfig{
			DefaultMode: "auto+",
			Predictor: PredictorConfig{
				EnableKnowledgeBase:   true,
				ConfidenceThreshold:   0.8,
				EnableExploration:     true,
				ExplorationCandidates: 3,
			},
			Formats: FormatsConfig{
				PNG: PNGFormatConfig{
					Target:          "jxl",
					Lossless:        true,
					Distance:        0,
					Effort:          7,
					EffortLargeFile: 5,
					EffortSmallFile: 9,
					LargeFileSizeMB: 10,
					SmallFileSizeKB: 100,
				},
				JPEG: JPEGFormatConfig{
					Target:       "jxl",
					LosslessJPEG: true,
					Effort:       7,
				},
				GIF: GIFFormatConfig{
					StaticTarget:   "jxl",
					AnimatedTarget: "avif",
					StaticDistance: 0,
					AnimatedCRF:    30,
					AnimatedSpeed:  6,
				},
				WebP: WebPFormatConfig{
					StaticTarget:   "jxl",
					AnimatedTarget: "avif",
				},
				Video: VideoFormatConfig{
					Target:         "mov",
					RepackageOnly:  true,
					EnableReencode: false,
					CRF:            23,
				},
			},
			QualityThresholds: QualityThresholdsConfig{
				Enable: true,
				Image: QualityThresholds{
					HighQuality:   2.0,
					MediumQuality: 0.5,
					LowQuality:    0.1,
				},
				Photo: QualityThresholds{
					HighQuality:   3.0,
					MediumQuality: 1.0,
					LowQuality:    0.1,
				},
				Animation: QualityThresholds{
					HighQuality:   20.0,
					MediumQuality: 1.0,
					LowQuality:    0.1,
				},
				Video: QualityThresholds{
					HighQuality:   100.0,
					MediumQuality: 10.0,
					LowQuality:    1.0,
				},
			},
			SupportedFormats: SupportedFormatsConfig{
				Image: []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".heic", ".heif"},
				Video: []string{".mp4", ".avi", ".mkv", ".mov", ".flv", ".m4v", ".3gp"},
			},
			ExcludedFormats: []string{".jxl", ".avif"},
		},

		Output: OutputConfig{
			KeepOriginal:              false,
			GenerateReport:            true,
			GeneratePerformanceReport: true,
			ReportFormat:              "both",
			FilenameTemplate:          "",
			DirectoryTemplate:         "",
		},

		Security: SecurityConfig{
			EnablePathCheck: true,
			ForbiddenDirectories: []string{
				"/System",
				"/Library",
				"/usr",
				"/bin",
				"/sbin",
				"/etc",
				"/var",
				"/tmp",
				"/private",
				"/Applications",
			},
			AllowedDirectories: []string{},
			CheckDiskSpace:     true,
			MinFreeSpaceMB:     1024,
			MaxFileSizeMB:      10240,
			EnableBackup:       true,
		},

		ProblemFiles: ProblemFilesConfig{
			CorruptedStrategy:             "skip",
			CodecIncompatibleStrategy:     "skip",
			ContainerIncompatibleStrategy: "skip",
			TrashStrategy:                 "delete",
			TrashExtensions:               []string{".tmp", ".bak", ".old", ".cache", ".log", ".db"},
			TrashKeywords:                 []string{"temp", "cache", "backup", "old", "trash"},
		},

		Resume: ResumeConfig{
			Enable:            true,
			SaveInterval:      10,
			AutoResumeOnCrash: false,
			PromptUser:        true,
		},

		UI: UIConfig{
			Mode:               "interactive",
			Theme:              "dark",
			EnableEmoji:        true,
			EnableASCIIArt:     true,
			EnableAnimations:   true,
			AnimationIntensity: "normal",
			Colors: UIColorsConfig{
				Primary:   "#00ff9f",
				Secondary: "#bd93f9",
				Success:   "#50fa7b",
				Warning:   "#ffb86c",
				Error:     "#ff5555",
				Info:      "#8be9fd",
			},
			Progress: ProgressUIConfig{
				RefreshIntervalMS: 100,
				AntiFlicker:       true,
				ShowFileIcons:     true,
				ShowETA:           true,
			},
			MonitorPanel: MonitorPanelConfig{
				Enable:           true,
				Position:         "top",
				RefreshIntervalS: 3,
				ShowCharts:       false,
			},
		},

		Logging: LoggingConfig{
			Level:      "info",
			Output:     "file",
			FilePath:   "~/.pixly/logs/pixly.log",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 7,
			Compress:   true,
		},

		Tools: ToolsConfig{
			AutoDetect:   true,
			CJXLPath:     "",
			DJXLPath:     "",
			AVIFEncPath:  "",
			AVIFDecPath:  "",
			FFmpegPath:   "",
			FFprobePath:  "",
			ExifToolPath: "",
		},

		Knowledge: KnowledgeConfig{
			Enable:        true,
			DBPath:        "~/.pixly/knowledge.db",
			AutoLearn:     true,
			MinConfidence: 0.8,
			Analysis: AnalysisConfig{
				Enable:          true,
				ReportInterval:  100,
				ShowSuggestions: true,
			},
		},

		Advanced: AdvancedConfig{
			EnableExperimental: false,
			EnableDebug:        false,
			MemoryPool: MemoryPoolConfig{
				Enable:       true,
				BufferSizeMB: 64,
			},
			Validation: ValidationConfig{
				EnablePixelCheck: false,
				EnableHashCheck:  false,
				MagicByteCheck:   true,
				SizeRatioCheck:   true,
				MaxSizeRatio:     1.5,
			},
		},

		Language: LanguageConfig{
			Default:    "zh_CN",
			AutoDetect: true,
		},

		Update: UpdateConfig{
			AutoCheck:         true,
			CheckIntervalDays: 7,
			NotifyOnUpdate:    true,
		},
	}
}
