/*
 * @Author: ybc
 * @Date: 2020-07-24 10:53:30
 * @Description: 缓存模块
 */
package services

import (
	"errors"
	"sync"
	"time"
)

// XwTable 是一个进程内缓存。三个 map 必须只在 Lock 保护下访问。
type XwTable struct {
	StringInt map[string]int64
	Strings   map[string]string
	StringMap map[string]*stringValue
	Lock      sync.Mutex
	Config    *XwTableConfig
	Tick      <-chan time.Time
}

type XwTableConfig struct {
	ClearIntervalTime time.Duration
}

type stringValue struct {
	Expire      int64
	AddTime     int64
	UpdatedTime int64
	Type        TableType
}

type TableType string

type stringMapArgs struct {
	Expire int64
	Type   TableType
}

const (
	STRING_INT TableType = "int"
	STRINGS    TableType = "string"
)

func NewXwTable() *XwTable {
	table := &XwTable{
		StringInt: make(map[string]int64),
		Strings:   make(map[string]string),
		StringMap: make(map[string]*stringValue),
		Config: &XwTableConfig{
			ClearIntervalTime: 3600,
		},
	}
	table.Tick = time.Tick(table.Config.ClearIntervalTime * time.Second)
	go table.HandleTick()

	return table
}

// HandleTick 在同一把锁内扫描并删除，避免遍历 map 时与读写 goroutine 并发。
func (this *XwTable) HandleTick() {
	for range this.Tick {
		this.Lock.Lock()
		for key := range this.StringMap {
			if this.keyIsExpireLocked(key) {
				this.delKeyLocked(key)
			}
		}
		this.Lock.Unlock()
	}
}

func (this *XwTable) DelKey(key string) error {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	return this.delKeyLocked(key)
}

func (this *XwTable) delKeyLocked(key string) error {
	_, ok := this.StringMap[key]
	if !ok {
		return errors.New("key:" + key + " not fund")
	}
	this.clearKeyLocked(key)
	return nil
}

func (this *XwTable) Incrby(key string, num int64) int64 {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.keyIsExpireLocked(key) {
		this.clearKeyLocked(key)
	}
	delete(this.Strings, key)
	this.StringInt[key] += num
	this.renewValueLocked(key, &stringMapArgs{Type: STRING_INT})
	return this.StringInt[key]
}

func (this *XwTable) SetExString(key string, expire int64, val string) bool {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.keyIsExpireLocked(key) {
		this.clearKeyLocked(key)
	}
	delete(this.StringInt, key)
	this.Strings[key] = val
	this.renewValueLocked(key, &stringMapArgs{Type: STRINGS, Expire: expire})
	return true
}

func (this *XwTable) SetExInt(key string, expire int64, val int64) bool {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.keyIsExpireLocked(key) {
		this.clearKeyLocked(key)
	}
	delete(this.Strings, key)
	this.StringInt[key] = val
	this.renewValueLocked(key, &stringMapArgs{Type: STRING_INT, Expire: expire})
	return true
}

func (this *XwTable) Expire(key string, expire int64) error {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if _, ok := this.StringMap[key]; !ok {
		return errors.New("key:" + key + " not fund")
	}
	this.renewValueLocked(key, &stringMapArgs{Expire: expire})
	return nil
}

func (this *XwTable) GetInt(key string) int64 {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.keyIsExpireLocked(key) {
		this.clearKeyLocked(key)
	}
	return this.StringInt[key]
}

func (this *XwTable) GetString(key string) string {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.keyIsExpireLocked(key) {
		this.clearKeyLocked(key)
	}
	return this.Strings[key]
}

func (this *XwTable) KeyIsExpire(key string) bool {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	return this.keyIsExpireLocked(key)
}

func (this *XwTable) keyIsExpireLocked(key string) bool {
	value, ok := this.StringMap[key]
	if !ok || value.Expire == 0 {
		return false
	}
	return time.Now().Unix()-(value.AddTime+value.Expire) > 0
}

func (this *XwTable) clearKeyLocked(key string) {
	delete(this.StringInt, key)
	delete(this.Strings, key)
	delete(this.StringMap, key)
}

func (this *XwTable) renewValueLocked(key string, args *stringMapArgs) {
	addTime := time.Now().Unix()
	if value, ok := this.StringMap[key]; ok {
		addTime = value.AddTime
		if args.Expire == 0 {
			args.Expire = value.Expire
		}
		if args.Type == "" {
			args.Type = value.Type
		}
	}
	this.StringMap[key] = &stringValue{
		Expire:      args.Expire,
		AddTime:     addTime,
		UpdatedTime: time.Now().Unix(),
		Type:        args.Type,
	}
}
