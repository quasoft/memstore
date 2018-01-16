package memstore

import "sync"

func newCache() cache {
	return cache{
		data: make(map[string]ValueType),
	}
}

type cache struct {
	data  map[string]ValueType
	mutex sync.RWMutex
}

func (c *cache) value(name string) (ValueType, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	v, ok := c.data[name]
	return v, ok
}

func (c *cache) setValue(name string, value ValueType) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[name] = value
}

func (c *cache) delete(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.data[name]; ok {
		delete(c.data, name)
	}
}
