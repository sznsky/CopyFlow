// Package logger 提供全局 zap 日志实例。
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger 结构化 logger（高性能）
	Logger *zap.Logger
	// Sugar Sugared logger（易用，性能略低）
	Sugar *zap.SugaredLogger
)

// Init 初始化全局 logger。
func Init(mode string) error {
	var config zap.Config

	if mode == "release" || mode == "production" {
		// 生产环境：JSON 格式，INFO 级别
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		config.Encoding = "json"
	} else {
		// 开发环境：彩色控制台输出，DEBUG 级别
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	// 自定义时间格式（ISO8601）
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	// 自定义调用者格式（短路径）
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var err error
	Logger, err = config.Build(
		zap.AddCaller(),      // 显示调用位置
		zap.AddCallerSkip(1), // 跳过 wrapper 层
		zap.AddStacktrace(zapcore.ErrorLevel), // ERROR 级别显示堆栈
	)
	if err != nil {
		return err
	}

	Sugar = Logger.Sugar()
	return nil
}

// Sync 刷新日志缓冲（程序退出前调用）。
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// 以下为便捷方法（使用 Sugared API）

// Debug 调试日志。
func Debug(msg string, keysAndValues ...interface{}) {
	Sugar.Debugw(msg, keysAndValues...)
}

// Info 信息日志。
func Info(msg string, keysAndValues ...interface{}) {
	Sugar.Infow(msg, keysAndValues...)
}

// Warn 警告日志。
func Warn(msg string, keysAndValues ...interface{}) {
	Sugar.Warnw(msg, keysAndValues...)
}

// Error 错误日志。
func Error(msg string, keysAndValues ...interface{}) {
	Sugar.Errorw(msg, keysAndValues...)
}

// Fatal 致命错误日志（会退出程序）。
func Fatal(msg string, keysAndValues ...interface{}) {
	Sugar.Fatalw(msg, keysAndValues...)
	os.Exit(1)
}

// With 添加固定字段（返回新的 logger）。
func With(keysAndValues ...interface{}) *zap.SugaredLogger {
	return Sugar.With(keysAndValues...)
}
