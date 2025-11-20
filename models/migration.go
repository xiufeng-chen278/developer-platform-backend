package models

import (
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

// Migration 用于记录已执行的版本。
type Migration struct {
	Version     int       `gorm:"primaryKey"`
	Description string    `gorm:"size:255"`
	AppliedAt   time.Time `gorm:"autoCreateTime"`
}

// MigrationItem 定义单个版本的执行逻辑。
type MigrationItem struct {
	Version     int
	Description string
	Up          func(tx *gorm.DB) error
	Down        func(tx *gorm.DB) error
}

// MigrationStatus 提供已执行及待执行版本的概览。
type MigrationStatus struct {
	Applied []Migration
	Pending []MigrationItem
}

// migrations 使用字面量静态定义，编译期即确定，不在运行时 append 或动态加载。
var migrations = []MigrationItem{
	{
		Version:     1,
		Description: "create_users_table",
		Up: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&User{})
		},
		Down: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&User{})
		},
	},
	{
		Version:     2,
		Description: "add_user_level_column",
		Up: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&User{}, "Level") {
				if err := tx.Migrator().AddColumn(&User{}, "Level"); err != nil {
					return err
				}
			}
			return tx.Model(&User{}).Where("level = 0").Update("level", DefaultUserLevel).Error
		},
		Down: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&User{}, "Level") {
				return tx.Migrator().DropColumn(&User{}, "Level")
			}
			return nil
		},
	},
	{
		Version:     3,
		Description: "create_api_keys_table",
		Up: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&APIKey{})
		},
		Down: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&APIKey{})
		},
	},
}

// RunMigrations 以幂等方式执行所有迁移。
func RunMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db 未初始化")
	}

	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for _, item := range migrations {
		var record Migration
		err := db.Where("version = ?", item.Version).First(&record).Error
		if err == nil {
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if item.Up == nil {
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := item.Up(tx); err != nil {
				return err
			}
			return tx.Create(&Migration{
				Version:     item.Version,
				Description: item.Description,
				AppliedAt:   time.Now(),
			}).Error
		}); err != nil {
			return fmt.Errorf("执行迁移 %d 失败: %w", item.Version, err)
		}
	}

	return nil
}

// RollbackMigration 回滚到指定版本。
func RollbackMigration(db *gorm.DB, version int) error {
	if db == nil {
		return fmt.Errorf("db 未初始化")
	}

	item, ok := findMigration(version)
	if !ok {
		return ErrMigrationNotFound
	}
	if item.Down == nil {
		return fmt.Errorf("迁移 %d 不支持回滚", version)
	}

	var record Migration
	if err := db.Where("version = ?", version).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrMigrationNotFound
		}
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := item.Down(tx); err != nil {
			return err
		}
		return tx.Delete(&Migration{}, "version = ?", version).Error
	})
}

// GetMigrationStatus 返回执行情况。
func GetMigrationStatus(db *gorm.DB) (*MigrationStatus, error) {
	if db == nil {
		return nil, fmt.Errorf("db 未初始化")
	}

	if err := ensureMigrationTable(db); err != nil {
		return nil, err
	}

	var applied []Migration
	if err := db.Order("version asc").Find(&applied).Error; err != nil {
		return nil, err
	}

	pending := make([]MigrationItem, 0)
	for _, item := range migrations {
		found := false
		for _, record := range applied {
			if record.Version == item.Version {
				found = true
				break
			}
		}
		if !found {
			pending = append(pending, item)
		}
	}

	return &MigrationStatus{
		Applied: applied,
		Pending: pending,
	}, nil
}

func ensureMigrationTable(db *gorm.DB) error {
	return db.AutoMigrate(&Migration{})
}

func findMigration(version int) (MigrationItem, bool) {
	for _, m := range migrations {
		if m.Version == version {
			return m, true
		}
	}
	return MigrationItem{}, false
}
