// Package runtimemodels holds model provider connections supplied by Command
// Center. Credentials and endpoints live only in the worker process.
package runtimemodels

import "sync"

type Connection struct {
	Provider      string `json:"provider"`
	BaseURL       string `json:"baseUrl,omitempty"`
	APIKey        string `json:"apiKey,omitempty"`
	PromptCaching bool   `json:"promptCaching"`
}

var store = struct {
	sync.RWMutex
	connections map[string]Connection
}{connections: make(map[string]Connection)}

func Replace(connections map[string]Connection) {
	store.Lock()
	defer store.Unlock()
	store.connections = make(map[string]Connection, len(connections))
	for name, connection := range connections {
		store.connections[name] = connection
	}
}

func Get(name string) (Connection, bool) {
	store.RLock()
	defer store.RUnlock()
	connection, ok := store.connections[name]
	return connection, ok
}

func LoadAll() map[string]Connection {
	store.RLock()
	defer store.RUnlock()
	result := make(map[string]Connection, len(store.connections))
	for name, connection := range store.connections {
		result[name] = connection
	}
	return result
}
