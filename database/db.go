package database

import (
	"database/sql"
	"fmt"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Init mysql
func Open(dns string, opts ...Option) (*gorm.DB, error) {
	o := defaultOptions()
	o.apply(opts...)

	sqlDB, err := sql.Open("mysql", dns)

	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(o.maxIdleConns)       // set the maximum number of connections in the idle connection pool
	sqlDB.SetMaxOpenConns(o.maxOpenConns)       // set the maximum number of open database connections
	sqlDB.SetConnMaxLifetime(o.connMaxLifetime) // set the maximum time a connection can be reused

	db, err := gorm.Open(mysqlDriver.New(mysqlDriver.Config{Conn: sqlDB}), gormConfig(o))
	if err != nil {
		return nil, fmt.Errorf("gorm.Open error, err: %w", err)
	}
	db.Set("gorm:table_options", "CHARSET=utf8mb4") // automatic appending of table suffixes when creating tables

	return db, nil
}

// gorm setting
func gormConfig(o *options) *gorm.Config {
	config := &gorm.Config{
		// disable foreign key constraints, not recommended for production environments
		DisableForeignKeyConstraintWhenMigrating: o.disableForeignKey,
		// removing the plural of an epithet
		NamingStrategy:         schema.NamingStrategy{SingularTable: true},
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	}

	// print all SQL
	if o.enableLogin {
		config.Logger = &logger{logger: o.logger, slowThreshold: o.slowThreshold}
	}

	return config
}
