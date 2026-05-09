package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type schemaMigrationRecord struct {
	ID          uint   `gorm:"primaryKey"`
	Version     string `gorm:"size:120;uniqueIndex;not null"`
	Description string `gorm:"size:255"`
	AppliedAt   int64  `gorm:"autoCreateTime:milli"`
}

type migrationStep struct {
	Version     string
	Description string
	Apply       func(db *gorm.DB) error
}

func runMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigrationRecord{}); err != nil {
		return err
	}

	steps := []migrationStep{
		{
			Version:     "20260509_001_plugin_type_column",
			Description: "Ensure plugin_type column exists on plugin records",
			Apply: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&PluginRecord{}, "plugin_type") {
					return nil
				}
				return tx.Migrator().AddColumn(&PluginRecord{}, "PluginType")
			},
		},
		{
			Version:     "20260509_002_plugin_config_unique_index",
			Description: "Ensure plugin config unique index is plugin_key + config_key",
			Apply: func(tx *gorm.DB) error {
				if tx.Migrator().HasIndex(&PluginConfigRecord{}, "idx_plugin_config_key") {
					if err := tx.Migrator().DropIndex(&PluginConfigRecord{}, "idx_plugin_config_key"); err != nil {
						return err
					}
				}
				if !tx.Migrator().HasIndex(&PluginConfigRecord{}, "uk_plugin_config_key") {
					if err := tx.Migrator().CreateIndex(&PluginConfigRecord{}, "uk_plugin_config_key"); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}

	for _, step := range steps {
		applied, err := isMigrationApplied(db, step.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := step.Apply(tx); err != nil {
				return err
			}
			row := schemaMigrationRecord{
				Version:     step.Version,
				Description: step.Description,
				AppliedAt:   time.Now().UnixMilli(),
			}
			return tx.Create(&row).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s failed: %w", step.Version, err)
		}
	}

	return nil
}

func isMigrationApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	if err := db.Model(&schemaMigrationRecord{}).Where("version = ?", version).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
