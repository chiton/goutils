package log

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// logContextKey is an unexported type and value used as a key to lookup values in FromContext().
// It is initialized to use the name of the zap.SugaredLogger type as a convention for other packages
// to be able to look up the context.
var logContextKey string

// FilePathEnvVar is an environment variable. If it is set in the executing environment, logs
// will also be written to the file specified in the value.
const FilePathEnvVar = "IAM_LOGFILE"

/*
DefaultLogger is the default logger. It is meant to be a conveniently accessible
logger for use in unit tests and dev-only environments. Please use NewLogger() for any code
that has the slightest possibility of making it into a production env.
*/
var DefaultLogger *zap.SugaredLogger

func init() {
	logType := reflect.TypeOf(zap.SugaredLogger{})
	logContextKey = logType.Name()

	logger, _ := zap.NewDevelopment()
	DefaultLogger = logger.Sugar()
}

/*
NewLogger returns a new sugared logger. If it detects that it is running in ECS, then it will activate
in production mode, else development mode. Additionally, detects the existence of a path in the env var
environ.FilePathEnvVar and will use that as the location of a file sink as well.
*/
func NewLogger(name string, production bool) (*zap.SugaredLogger, error) {
	var (
		logger    *zap.Logger
		err       error
		logConfig zap.Config
	)

	if production {
		logConfig = zap.NewProductionConfig()
		logConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(strconv.FormatInt(t.UnixNano(), 10))
		}
	} else {
		logConfig = zap.NewDevelopmentConfig()

		logFilePath := os.Getenv(FilePathEnvVar)
		logfileDir := filepath.Dir(logFilePath)

		err = os.MkdirAll(logfileDir, 0650)
		if err != nil {
			return nil, err
		}

		if logFilePath != "" {
			logConfig.OutputPaths = append(logConfig.OutputPaths, logFilePath)
		}
	}

	logConfig.DisableStacktrace = true
	logger, err = logConfig.Build()

	if err != nil {
		return nil, err
	}

	sugar := logger.Sugar()
	sugar = sugar.Named(name)

	if production {
		sugar.Info("Production environment detected -- logging in production mode")
	} else {
		sugar.Info("Logging in development mode")
	}

	return sugar, nil
}

// FromContext returns a zap.SugaredLogger from the given context. If it doesn't find one,
// it will return the DefaultLogger.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(logContextKey).(*zap.SugaredLogger); ok {
		return logger
	}

	// DefaultLogger.Warn("Logger not found in context")

	return DefaultLogger
}

// NewContext creates a new context with the given logger
func NewContext(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, logContextKey, logger)
}
