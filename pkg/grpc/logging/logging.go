package logging

import (
	"context"

	grpc_log "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type grpcLogger struct {
	logging.Logger
}

func (l grpcLogger) Log(_ context.Context, level grpc_log.Level, msg string, fields ...any) {
	lw := l.SugaredLogger().With(fields)
	switch level {
	case grpc_log.LevelError:
		lw.Error(msg)
	case grpc_log.LevelWarn:
		lw.Warn(msg)
	case grpc_log.LevelInfo:
		lw.Info(msg)
	case grpc_log.LevelDebug:
		lw.Debug(msg)
	}
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that logs incoming requests
// and associated tags to "logger".
func UnaryServerInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor {
	return grpc_log.UnaryServerInterceptor(grpcLogger{logger}, grpc_log.WithLevels(grpc_log.DefaultServerCodeToLevel))
}

// InitGrpcLogger initializes gRPC logger using our logging framework.
func InitGrpcLogger() {
	module := logging.ModuleForName("grpc_internal")
	// Skipping 4 nested levels to show correct place in code where issue happened.
	// This is due to the way gRPC library wraps logger.
	l := logging.CreateLogger(module, 4)
	grpclog.SetLoggerV2(&zapGrpcLogger{
		logger: l,
	})
}

type zapGrpcLogger struct {
	logger logging.Logger
}

func (l *zapGrpcLogger) Info(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *zapGrpcLogger) Infoln(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *zapGrpcLogger) Infof(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *zapGrpcLogger) Warning(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *zapGrpcLogger) Warningln(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *zapGrpcLogger) Warningf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *zapGrpcLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *zapGrpcLogger) Errorln(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *zapGrpcLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *zapGrpcLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *zapGrpcLogger) Fatalln(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *zapGrpcLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *zapGrpcLogger) V(level int) bool {
	return level <= 0
}
