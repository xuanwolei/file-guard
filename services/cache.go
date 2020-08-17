/*
 * @Author: ybc
 * @Date: 2020-07-24 10:53:30
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-17 14:56:08
 * @Description: 缓存模块
 */
package services

import (
	"errors"
	"sync"
	"time"
)

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
		StringMap: make(map[string]*stringValue),
		Config: &XwTableConfig{
			ClearIntervalTime: 3600,
		},
	}
	table.Tick = time.Tick(table.Config.ClearIntervalTime * time.Second)
	go table.HandleTick()

	return table
}

//定时清除过期的key
func (this *XwTable) HandleTick() {
	for {
		select {
		case <-this.Tick:
			var num int
			for k, _ := range this.StringMap {
				if this.KeyIsExpire(k) {
					this.DelKey(k)
					num++
				}
			}
		}
	}
}

func (this *XwTable) DelKey(key string) error {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.StringMap[key] == nil {
		return errors.New("key:" + key + " not fund")
	}
	switch this.StringMap[key].Type {
	case STRING_INT:
		delete(this.StringInt, key)
	case STRINGS:
		delete(this.Strings, key)

	}
	delete(this.StringMap, key)
	return nil
}

func (this *XwTable) Incrby(key string, num int64) int64 {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.KeyIsExpire(key) {
		this.resetStringInt(key)
	}
	this.StringInt[key] += num
	this.renewValue(key, &stringMapArgs{
		Type: STRING_INT,
	})
	return this.StringInt[key]
}

func (this *XwTable) SetExString(key string, expire int64, val string) bool {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.KeyIsExpire(key) {
		this.resetStrings(key)
	}
	this.Strings[key] = val
	this.renewValue(key, &stringMapArgs{
		Type:   STRINGS,
		Expire: expire,
	})

	return true
}

func (this *XwTable) SetExInt(key string, expire int64, val int64) bool {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.KeyIsExpire(key) {
		this.resetStringInt(key)
	}
	this.StringInt[key] = val
	this.renewValue(key, &stringMapArgs{
		Type:   STRING_INT,
		Expire: expire,
	})

	return true
}

func (this *XwTable) Expire(key string, expire int64) error {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	this.renewValue(key, &stringMapArgs{
		Expire: expire,
	})
	return nil
}

func (this *XwTable) GetInt(key string) int64 {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	if this.KeyIsExpire(key) {
		this.resetStringInt(key)
	}
	return this.StringInt[key]
}

func (this *XwTable) GetString(key string) string {

	if this.KeyIsExpire(key) {
		this.Lock.Lock()
		defer this.Lock.Unlock()
		this.resetStrings(key)
	}
	return this.Strings[key]
}

func (this *XwTable) KeyIsExpire(key string) bool {

	if this.StringMap[key] == nil {
		return false
	}
	if this.StringMap[key].Expire == 0 {
		return false
	}
	if time.Now().Unix()-(this.StringMap[key].AddTime+this.StringMap[key].Expire) > 0 {
		return true
	}

	return false
}

func (this *XwTable) resetStringInt(key string) {
	delete(this.StringInt, key)
	this.resetStringMap(key)
	return
}

func (this *XwTable) resetStrings(key string) {
	delete(this.Strings, key)
	this.resetStringMap(key)
	return
}

func (this *XwTable) resetStringMap(key string) {
	delete(this.StringMap, key)
	return
}

func (this *XwTable) renewValue(key string, args *stringMapArgs) {
	addTime := time.Now().Unix()
	if this.StringMap[key] != nil {
		addTime = this.StringMap[key].AddTime
		if args.Expire == 0 {
			args.Expire = this.StringMap[key].Expire
		}
	}

	this.StringMap[key] = &stringValue{
		Expire:      args.Expire,
		AddTime:     addTime,
		UpdatedTime: time.Now().Unix(),
		Type:        args.Type,
	}

	return
}
