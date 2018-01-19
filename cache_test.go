package memstore

import (
	"reflect"
	"testing"
)

func Test_newCache(t *testing.T) {
	cache := newCache()
	if cache == nil {
		t.Error("newCache() got nil")
	}
}

func Test_cache_value(t *testing.T) {
	cache := newCache()
	cache.setValue("key1", valueType{"subkey1": "value1"})
	cache.setValue("key2", nil)
	cache.setValue("key3", valueType{"subkey3": nil})

	tests := []struct {
		name   string
		key    string
		want   valueType
		wantOk bool
	}{
		{"Existing key", "key1", valueType{"subkey1": "value1"}, true},
		{"Existing key with nil value type", "key2", nil, true},
		{"Existing key and subkey with nil value", "key3", valueType{"subkey3": nil}, true},
		{"Not existing key", "thereisnokey", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := cache.value(tt.key)

			if gotOk != tt.wantOk {
				t.Errorf("cache.value(%v) got ok = %v, want %v", tt.key, gotOk, tt.wantOk)
			}

			if gotOk && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cache.value(%v) got = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func Test_cache_delete(t *testing.T) {
	cache := newCache()

	cache.setValue("key1", valueType{"subkey1": "value1"})

	_, gotOk := cache.value("key1")
	if !gotOk {
		t.Error(`cache.value("key1") got ok = false, want true`)
	}

	cache.delete("key1")

	_, gotOk = cache.value("key1")
	if gotOk {
		t.Error(`cache.value("key1") got ok = true, want false`)
	}
}
