package serverInfo

import (
	"os"

	"github.com/sunvc/NoLets/common"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger 初始化日志组件
func InitLogger(path string) *zap.Logger {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    50, // 2核1G机器，50MB比较合适
		MaxBackups: 3,  // 保留3个，省磁盘
		MaxAge:     7,  // 只留一周
		Compress:   true,
		LocalTime:  true,
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	// Loki 喜欢 ISO8601 格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder // info, error (小写更标准)

	var core zapcore.Core
	if common.LocalConfig.System.Debug {
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(lumberJackLogger), zap.InfoLevel),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(lumberJackLogger),
			zap.InfoLevel,
		)
	}

	// 重点：加上固定字段，方便 Loki 分类
	return zap.New(core, zap.AddCaller(), zap.Fields(
		zap.String("service", "nolets"),
	))
}
