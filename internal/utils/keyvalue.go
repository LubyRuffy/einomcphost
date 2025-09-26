package utils

import (
	"sort"
	"sync"
)

// KeyValue 是一个键值对元素，用于存储统计信息，提供排序功能
type KeyValue struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

// KeyValueMap 是一个键值对，用于存储统计信息
type KeyValueMap struct {
	sync.Map
}

// Add 添加一个键值对，如果键不存在，则创建一个新键值对，如果键存在，则将值加1
func (m *KeyValueMap) Add(key string, f func()) {
	count, _ := m.LoadOrStore(key, 0)
	m.Store(key, count.(int)+1)
	if f != nil {
		f()
	}
}

// GetSorted 返回按照值排序后的键值对列表
func (m *KeyValueMap) GetSorted() []*KeyValue {
	items := make([]*KeyValue, 0)
	m.Range(func(key, value interface{}) bool {
		items = append(items, &KeyValue{Key: key.(string), Value: value.(int)})
		return true
	})
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})
	return items
}

func (m *KeyValueMap) SetValues(v []*KeyValue, f func()) {
	m.Clear()
	for _, item := range v {
		m.Store(item.Key, item.Value)
	}
	if f != nil {
		f()
	}
}
