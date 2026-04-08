package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// Logger slog 日志包装器，提供类似 zap 的 API
type Logger struct {
	logger *slog.Logger
}

// NewLogger 创建新的 Logger 实例
func NewLogger(level string, format string, output string, filePath string) (*Logger, error) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	var out io.Writer

	// 确定输出目标
	if output == "file" && filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		out = f
	} else {
		out = os.Stdout
	}

	// 确定格式
	if format == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	return &Logger{
		logger: slog.New(handler),
	}, nil
}

// GetSlogger 获取底层的 slog.Logger
func (l *Logger) GetSlogger() *slog.Logger {
	return l.logger
}

// Debug 记录调试日志
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info 记录信息日志
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn 记录警告日志
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error 记录错误日志
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Fatal 记录致命错误并退出程序
func (l *Logger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

// Sync 刷新日志缓冲区（slog 不需要，但为了兼容 zap 接口）
func (l *Logger) Sync() error {
	return nil
}

// With 创建带有额外字段的子 Logger
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
	}
}

// Helper functions for common log fields

// String 创建字符串字段
func String(key string, value string) slog.Attr {
	return slog.String(key, value)
}

// Int 创建整数字段
func Int(key string, value int) slog.Attr {
	return slog.Int(key, value)
}

// Int64 创建 64 位整数字段
func Int64(key string, value int64) slog.Attr {
	return slog.Int64(key, value)
}

// Float64 创建浮点数字段
func Float64(key string, value float64) slog.Attr {
	return slog.Float64(key, value)
}

// Bool 创建布尔字段
func Bool(key string, value bool) slog.Attr {
	return slog.Bool(key, value)
}

// Duration 创建持续时间段字段
func Duration(key string, value time.Duration) slog.Attr {
	return slog.Duration(key, value)
}

// Time 创建时间字段
func Time(key string, value time.Time) slog.Attr {
	return slog.Time(key, value)
}

// Any 创建任意类型字段
func Any(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

// Err 创建错误字段（zap 兼容）
func Err(err error) slog.Attr {
	if err != nil {
		return slog.Any("error", err)
	}
	return slog.Any("error", nil)
}

// Group 创建字段组
func Group(key string, args ...any) slog.Attr {
	return slog.Group(key, args...)
}

// WithContext 从 context 中提取 trace_id 等字段
func WithContext(ctx context.Context, args ...any) []any {
	// 可以从 context 中提取 trace_id 等信息
	// 这里简化处理，直接返回传入的参数
	return args
}
