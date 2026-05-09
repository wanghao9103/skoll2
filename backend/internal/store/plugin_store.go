package store

import (
	"encoding/json"
	"errors"

	"skoll2/backend/internal/plugin"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PluginStore struct {
	db *gorm.DB
}

type PluginRecord struct {
	ID            uint   `gorm:"primaryKey"`
	PluginKey     string `gorm:"size:120;uniqueIndex;not null"`
	Name          string `gorm:"size:120;not null"`
	Version       string `gorm:"size:40;not null"`
	Description   string `gorm:"type:text"`
	Icon          string `gorm:"size:120"`
	Status        string `gorm:"size:40;not null"`
	APIPrefix     string `gorm:"size:200"`
	FrontendEntry string `gorm:"size:255"`
	MenusJSON     string `gorm:"type:longtext"`
	PermsJSON     string `gorm:"type:longtext"`
}

type PluginConfigRecord struct {
	ID        uint   `gorm:"primaryKey"`
	PluginKey string `gorm:"size:120;index:idx_plugin_key"`
	ConfigKey string `gorm:"size:120;index:idx_plugin_config_key,unique"`
	Value     string `gorm:"type:text"`
	IsSecret  bool
}

type PluginConfigItem struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}

func NewPluginStore(driver string, dsn string) (*PluginStore, error) {
	if driver == "" {
		driver = "sqlite"
	}

	var (
		db  *gorm.DB
		err error
	)

	switch driver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	default:
		return nil, errors.New("unsupported DB_DRIVER, use mysql or sqlite")
	}
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&PluginRecord{}, &PluginConfigRecord{}); err != nil {
		return nil, err
	}

	return &PluginStore{db: db}, nil
}

func (s *PluginStore) ListPlugins() ([]plugin.Item, error) {
	var rows []PluginRecord
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]plugin.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromRecord(row))
	}
	return items, nil
}

func (s *PluginStore) UpsertPlugin(item plugin.Item) error {
	row := toRecord(item)
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "plugin_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "version", "description", "icon", "status", "api_prefix", "frontend_entry", "menus_json", "perms_json"}),
	}).Create(&row).Error
}

func (s *PluginStore) DeletePlugin(pluginKey string) error {
	if err := s.db.Where("plugin_key = ?", pluginKey).Delete(&PluginRecord{}).Error; err != nil {
		return err
	}
	if err := s.db.Where("plugin_key = ?", pluginKey).Delete(&PluginConfigRecord{}).Error; err != nil {
		return err
	}
	return nil
}

func (s *PluginStore) ListPluginConfigs(pluginKey string) ([]PluginConfigItem, error) {
	var rows []PluginConfigRecord
	if err := s.db.Where("plugin_key = ?", pluginKey).Find(&rows).Error; err != nil {
		return nil, err
	}

	configs := make([]PluginConfigItem, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, PluginConfigItem{
			Key:      row.ConfigKey,
			Value:    row.Value,
			IsSecret: row.IsSecret,
		})
	}
	return configs, nil
}

func (s *PluginStore) ReplacePluginConfigs(pluginKey string, configs []PluginConfigItem) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("plugin_key = ?", pluginKey).Delete(&PluginConfigRecord{}).Error; err != nil {
			return err
		}

		for _, cfg := range configs {
			if cfg.Key == "" {
				continue
			}
			row := PluginConfigRecord{
				PluginKey: pluginKey,
				ConfigKey: cfg.Key,
				Value:     cfg.Value,
				IsSecret:  cfg.IsSecret,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func toRecord(item plugin.Item) PluginRecord {
	menus, _ := json.Marshal(item.Menus)
	perms, _ := json.Marshal(item.Permissions)

	return PluginRecord{
		PluginKey:     item.Key,
		Name:          item.Name,
		Version:       item.Version,
		Description:   item.Description,
		Icon:          item.Icon,
		Status:        string(item.Status),
		APIPrefix:     item.APIPrefix,
		FrontendEntry: item.FrontendEntry,
		MenusJSON:     string(menus),
		PermsJSON:     string(perms),
	}
}

func fromRecord(row PluginRecord) plugin.Item {
	menus := make([]plugin.Menu, 0)
	perms := make([]string, 0)
	_ = json.Unmarshal([]byte(row.MenusJSON), &menus)
	_ = json.Unmarshal([]byte(row.PermsJSON), &perms)

	return plugin.Item{
		Name:          row.Name,
		Key:           row.PluginKey,
		Version:       row.Version,
		Description:   row.Description,
		Icon:          row.Icon,
		Status:        plugin.Status(row.Status),
		APIPrefix:     row.APIPrefix,
		FrontendEntry: row.FrontendEntry,
		Menus:         menus,
		Permissions:   perms,
	}
}
