package pluginruntime

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Runtime struct {
	db    *gorm.DB
	cache *Cache
}

func New(db *gorm.DB, cache *Cache) *Runtime {
	return &Runtime{db: db, cache: cache}
}

func (r *Runtime) DB() *DB {
	return &DB{db: r.db}
}

func (r *Runtime) Cache() *Cache {
	return r.cache
}

type DB struct {
	db *gorm.DB
}

func (d *DB) List(table string, order string) ([]map[string]any, error) {
	rows := make([]map[string]any, 0)
	q := d.db.Table(table)
	if order != "" {
		q = q.Order(order)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (d *DB) FirstByID(table string, id int64) (map[string]any, error) {
	row := map[string]any{}
	if err := d.db.Table(table).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (d *DB) Create(table string, values map[string]any) (map[string]any, error) {
	if err := d.db.Table(table).Create(&values).Error; err != nil {
		return nil, err
	}

	id := toInt64(values["id"])
	if id <= 0 {
		return values, nil
	}
	return d.FirstByID(table, id)
}

func (d *DB) UpdateByID(table string, id int64, values map[string]any) (map[string]any, error) {
	if err := d.db.Table(table).Where("id = ?", id).Updates(values).Error; err != nil {
		return nil, err
	}
	return d.FirstByID(table, id)
}

func (d *DB) DeleteByID(table string, id int64) error {
	return d.db.Table(table).Where("id = ?", id).Delete(nil).Error
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

type Cache struct {
	store sync.Map
}

type cacheEntry struct {
	value     any
	expiresAt int64
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Set(namespace string, key string, value any, ttlSeconds int64) bool {
	expiresAt := int64(0)
	if ttlSeconds > 0 {
		expiresAt = time.Now().Unix() + ttlSeconds
	}
	c.store.Store(c.joinKey(namespace, key), cacheEntry{value: value, expiresAt: expiresAt})
	return true
}

func (c *Cache) Get(namespace string, key string) (any, bool) {
	v, ok := c.store.Load(c.joinKey(namespace, key))
	if !ok {
		return nil, false
	}

	entry, ok := v.(cacheEntry)
	if !ok {
		return nil, false
	}

	if entry.expiresAt > 0 && time.Now().Unix() > entry.expiresAt {
		c.store.Delete(c.joinKey(namespace, key))
		return nil, false
	}

	return entry.value, true
}

func (c *Cache) Delete(namespace string, key string) bool {
	c.store.Delete(c.joinKey(namespace, key))
	return true
}

func (c *Cache) joinKey(namespace string, key string) string {
	return fmt.Sprintf("%s:%s", namespace, key)
}
