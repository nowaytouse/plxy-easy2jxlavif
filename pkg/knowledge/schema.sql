-- Pixly v3.0 知识库 Schema
-- 用于记录转换历史，优化预测准确性

-- ============================================================================
-- 1. 转换记录表 (核心)
-- ============================================================================
CREATE TABLE IF NOT EXISTS conversion_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 文件信息
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    original_format TEXT NOT NULL,  -- "png", "jpg", "gif" etc.
    original_size INTEGER NOT NULL, -- bytes
    
    -- 文件特征
    width INTEGER,
    height INTEGER,
    has_alpha BOOLEAN,
    pix_fmt TEXT,
    is_animated BOOLEAN DEFAULT 0,
    frame_count INTEGER DEFAULT 1,
    estimated_quality INTEGER,      -- 0-100
    
    -- 预测信息
    predictor_name TEXT,            -- "PNGPredictor", "JPEGPredictor" etc.
    prediction_rule TEXT,           -- "PNG_ALWAYS_JXL_LOSSLESS" etc.
    prediction_confidence REAL,     -- 0-1
    prediction_time_ms INTEGER,
    
    -- 预测参数
    predicted_format TEXT,          -- "jxl", "avif", "mov"
    predicted_lossless BOOLEAN,
    predicted_distance REAL,
    predicted_effort INTEGER,
    predicted_lossless_jpeg BOOLEAN,
    predicted_crf INTEGER,
    predicted_speed INTEGER,
    
    -- 预测的空间节省
    predicted_saving_percent REAL,  -- 0-1
    predicted_output_size INTEGER,  -- bytes
    
    -- 实际转换结果
    actual_format TEXT,
    actual_output_size INTEGER,     -- bytes
    actual_conversion_time_ms INTEGER,
    
    -- 实际空间节省
    actual_saving_percent REAL,     -- 0-1
    actual_saving_bytes INTEGER,
    
    -- 质量验证
    validation_method TEXT,         -- "pixel_diff", "bit_level", "psnr", "ssim"
    validation_passed BOOLEAN,
    pixel_diff_percent REAL,        -- 像素差异百分比 (0=完美)
    psnr_value REAL,                -- PSNR值 (越高越好)
    ssim_value REAL,                -- SSIM值 (0-1, 越接近1越好)
    
    -- 预测准确性
    prediction_error_percent REAL,  -- |predicted - actual| / actual
    was_explored BOOLEAN DEFAULT 0, -- 是否使用了探索引擎
    
    -- 用户反馈（可选）
    user_rating INTEGER,            -- 1-5星
    user_comment TEXT,
    
    -- 元数据
    pixly_version TEXT,
    host_os TEXT,
    
    UNIQUE(file_path, created_at)
);

-- 索引优化查询
CREATE INDEX IF NOT EXISTS idx_original_format ON conversion_records(original_format);
CREATE INDEX IF NOT EXISTS idx_predictor_name ON conversion_records(predictor_name);
CREATE INDEX IF NOT EXISTS idx_prediction_rule ON conversion_records(prediction_rule);
CREATE INDEX IF NOT EXISTS idx_created_at ON conversion_records(created_at);
CREATE INDEX IF NOT EXISTS idx_validation_passed ON conversion_records(validation_passed);

-- ============================================================================
-- 2. 预测准确性统计表
-- ============================================================================
CREATE TABLE IF NOT EXISTS prediction_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 统计维度
    predictor_name TEXT NOT NULL,
    prediction_rule TEXT NOT NULL,
    original_format TEXT NOT NULL,
    
    -- 统计时间范围
    stats_from DATE,
    stats_to DATE,
    
    -- 样本数量
    total_conversions INTEGER DEFAULT 0,
    successful_conversions INTEGER DEFAULT 0,
    
    -- 预测准确性
    avg_prediction_error_percent REAL,
    median_prediction_error_percent REAL,
    std_prediction_error_percent REAL,
    
    -- 空间节省统计
    avg_predicted_saving REAL,
    avg_actual_saving REAL,
    
    -- 质量统计
    perfect_quality_count INTEGER DEFAULT 0,  -- pixel_diff = 0 或 bit-level相同
    good_quality_count INTEGER DEFAULT 0,     -- PSNR > 40 或 SSIM > 0.95
    
    -- 时间统计
    avg_conversion_time_ms INTEGER,
    
    -- 更新时间
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(predictor_name, prediction_rule, original_format)
);

-- ============================================================================
-- 3. 异常案例表（用于特殊处理）
-- ============================================================================
CREATE TABLE IF NOT EXISTS anomaly_cases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 异常记录引用
    conversion_record_id INTEGER,
    
    -- 异常类型
    anomaly_type TEXT NOT NULL,  -- "large_error", "quality_issue", "format_mismatch" etc.
    anomaly_severity TEXT,       -- "low", "medium", "high"
    
    -- 异常描述
    description TEXT,
    
    -- 检测时间
    detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 是否已处理
    resolved BOOLEAN DEFAULT 0,
    resolution_note TEXT,
    
    FOREIGN KEY (conversion_record_id) REFERENCES conversion_records(id)
);

CREATE INDEX IF NOT EXISTS idx_anomaly_type ON anomaly_cases(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_resolved ON anomaly_cases(resolved);

-- ============================================================================
-- 4. 格式特征统计表（用于优化预测）
-- ============================================================================
CREATE TABLE IF NOT EXISTS format_characteristics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 格式标识
    original_format TEXT NOT NULL,
    pix_fmt TEXT,
    
    -- 尺寸范围
    size_range TEXT,  -- "tiny", "small", "medium", "large", "huge"
    
    -- 统计数据
    sample_count INTEGER DEFAULT 0,
    
    -- 最优转换策略
    best_target_format TEXT,
    best_avg_saving REAL,
    best_success_rate REAL,
    
    -- 更新时间
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(original_format, pix_fmt, size_range)
);

-- ============================================================================
-- 5. 视图：最新转换记录
-- ============================================================================
CREATE VIEW IF NOT EXISTS recent_conversions AS
SELECT 
    id,
    created_at,
    file_name,
    original_format,
    predicted_format,
    ROUND(original_size / 1024.0 / 1024.0, 2) as original_mb,
    ROUND(actual_output_size / 1024.0 / 1024.0, 2) as output_mb,
    ROUND(actual_saving_percent * 100, 1) as saving_percent,
    validation_passed,
    prediction_error_percent
FROM conversion_records
ORDER BY created_at DESC
LIMIT 100;

-- ============================================================================
-- 6. 视图：预测准确性汇总
-- ============================================================================
CREATE VIEW IF NOT EXISTS prediction_accuracy_summary AS
SELECT 
    predictor_name,
    prediction_rule,
    COUNT(*) as total,
    ROUND(AVG(prediction_error_percent), 2) as avg_error,
    ROUND(AVG(actual_saving_percent * 100), 1) as avg_saving,
    SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END) as passed_count,
    ROUND(100.0 * SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END) / COUNT(*), 1) as pass_rate
FROM conversion_records
GROUP BY predictor_name, prediction_rule
ORDER BY total DESC;

