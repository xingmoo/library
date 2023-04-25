package database

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
	"time"
)

var _ gormlogger.Interface = (*logger)(nil)

type logger struct {
	logger        *zap.Logger
	slowThreshold time.Duration
}

func (l *logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	//panic("implement me")
	return l
}

func (l *logger) Info(ctx context.Context, s string, i ...interface{}) {
	l.logger.Info(fmt.Sprintf(s, i))
}

func (l *logger) Warn(ctx context.Context, s string, i ...interface{}) {
	l.logger.Warn(fmt.Sprintf(s, i))
}

func (l *logger) Error(ctx context.Context, s string, i ...interface{}) {
	l.logger.Error(fmt.Sprintf(s, i))
}

func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {

	elapsed := time.Since(begin)
	sql, rows := fc()
	if l.slowThreshold != 0 && elapsed > l.slowThreshold {
		l.logger.Warn("slow query",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Time("begin", begin),
			zap.String("elapsed", elapsed.String()),
		)
		return
	}

	l.logger.Info("trace",
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Time("begin", begin),
		zap.String("elapsed", elapsed.String()),
	)
}
