// Package ordered provides ordered, deterministic traversal of maps.
package ordered

import (
	"reflect"
	"sort"
)

// Types implementing the Map interface can be traversed in
// deterministic order. The Keys method must return all keys
// in the map.
type Map interface {
	Keys() []string
}

func RangeMap(m Map, fn func(string)) {
	keys := m.Keys()
	sort.Strings(keys)
	for _, k := range keys {
		fn(k)
	}
}

// RangeStringMap calls fn on each string key in v in deterministic order. If
// v is not a map with a string for a key type, RangeStringMap panics.
func RangeStrings(v interface{}, fn func(string)) {
	val := reflect.ValueOf(v)
	keys := make([]string, 0, val.Len())
	for _, k := range val.MapKeys() {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	for _, k := range keys {
		fn(k)
	}
}
