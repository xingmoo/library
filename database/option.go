package database

import (
	"go.uber.org/zap"
	"time"
)

// Option set the mysql options.
type Option func(*options)

type options struct {
	maxIdleConns    int
	maxOpenConns    int
	connMaxLifetime time.Duration
	enableLogin     bool
	slowThreshold   time.Duration

	disableForeignKey bool

	logger *zap.Logger
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// default settings
func defaultOptions() *options {
	return &options{

		maxIdleConns:    3,                // set the maximum number of connections in the idle connection pool
		maxOpenConns:    50,               // set the maximum number of open database connections
		connMaxLifetime: 30 * time.Minute, // sets the maximum amount of time a connection can be reused

		disableForeignKey: true, // disables the use of foreign keys, true is recommended for production environments, enabled by default
	}
}

// WithMaxIdleConns set max idle conns
func WithMaxIdleConns(size int) Option {
	return func(o *options) {
		o.maxIdleConns = size
	}
}

// WithMaxOpenConns set max open conns
func WithMaxOpenConns(size int) Option {
	return func(o *options) {
		o.maxOpenConns = size
	}
}

// WithConnMaxLifetime set conn max lifetime
func WithConnMaxLifetime(t time.Duration) Option {
	return func(o *options) {
		o.connMaxLifetime = t
	}
}

// WithEnableForeignKey use foreign keys
func WithEnableForeignKey() Option {
	return func(o *options) {
		o.disableForeignKey = false
	}
}

// WithLog set log sql
func WithLog(b bool, zlogger *zap.Logger, slowThreshold time.Duration) Option {
	return func(o *options) {
		o.enableLogin = b
		o.logger = zlogger
		o.slowThreshold = slowThreshold
	}
}
