package dao

import (
	"fmt"
	"strings"

	"ragflow/internal/server"

	gormLogger "gorm.io/gorm/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openDatabase(dbCfg server.DatabaseConfig, serverMode string) (*gorm.DB, error) {
	var (
		dialector gorm.Dialector
		driver    = strings.ToLower(strings.TrimSpace(dbCfg.Driver))
	)

	switch driver {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
			dbCfg.Host,
			dbCfg.Username,
			dbCfg.Password,
			dbCfg.Database,
			dbCfg.Port,
		)
		dialector = postgres.Open(dsn)
	case "mysql", "":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			dbCfg.Username,
			dbCfg.Password,
			dbCfg.Host,
			dbCfg.Port,
			dbCfg.Database,
			defaultCharset(dbCfg.Charset),
		)
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", dbCfg.Driver)
	}

	var gormLogLevel gormLogger.LogLevel
	if serverMode == "debug" {
		gormLogLevel = gormLogger.Info
	} else {
		gormLogLevel = gormLogger.Silent
	}

	return gorm.Open(dialector, &gorm.Config{
		Logger:         gormLogger.Default.LogMode(gormLogLevel),
		NowFunc:        localNow,
		TranslateError: true,
	})
}

func defaultCharset(charset string) string {
	if strings.TrimSpace(charset) == "" {
		return "utf8mb4"
	}
	return charset
}

func isBenignMigrationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "already exists") {
		return true
	}
	if strings.Contains(errStr, "duplicate key") {
		return true
	}
	if strings.Contains(errStr, "duplicate column") {
		return true
	}
	if strings.Contains(errStr, "error 1061") && strings.Contains(errStr, "duplicate key name") {
		return true
	}
	if strings.Contains(errStr, "error 1060") && strings.Contains(errStr, "duplicate column name") {
		return true
	}
	if strings.Contains(errStr, "error 1050") && strings.Contains(errStr, "table") {
		return true
	}
	if strings.Contains(errStr, "error 1091") && strings.Contains(errStr, "can't drop") {
		return true
	}
	if strings.Contains(errStr, "error 1138") && strings.Contains(errStr, "invalid use of null") {
		return true
	}
	return false
}
