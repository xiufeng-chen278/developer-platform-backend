package models

import "errors"

var (
	// ErrMissingConfig 表示初始化数据库时缺少配置。
	ErrMissingConfig = errors.New("缺少配置")
	// ErrMigrationNotFound 在回滚指定版本时找不到对应记录。
	ErrMigrationNotFound = errors.New("未找到指定迁移版本")
)
