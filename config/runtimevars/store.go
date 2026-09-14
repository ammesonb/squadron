// Package runtimevars holds the workspace variable snapshot supplied by the
// Command Center. Values live only in process memory on the Squadron worker.
package runtimevars

import (
	"fmt"
	"sync"
)

var store = struct {
	sync.RWMutex
	values map[string]string
}{values: make(map[string]string)}

func Replace(values map[string]string) {
	store.Lock()
	defer store.Unlock()
	store.values = make(map[string]string, len(values))
	for name, value := range values {
		store.values[name] = value
	}
}

func LoadAll() map[string]string {
	store.RLock()
	defer store.RUnlock()
	values := make(map[string]string, len(store.values))
	for name, value := range store.values {
		values[name] = value
	}
	return values
}

func Get(name string) (string, error) {
	store.RLock()
	defer store.RUnlock()
	value, ok := store.values[name]
	if !ok {
		return "", fmt.Errorf("variable %q not found", name)
	}
	return value, nil
}

// Set and Delete support short-lived runtime credentials such as an MCP OAuth
// refresh. Durable credential ownership belongs to Command Center.
func Set(name, value string) {
	store.Lock()
	defer store.Unlock()
	store.values[name] = value
}

func Delete(name string) {
	store.Lock()
	defer store.Unlock()
	delete(store.values, name)
}
