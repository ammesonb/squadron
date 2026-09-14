package wsbridge

import (
	"sync"

	"squadron/config"
)

// MissionConcurrencyTracker enforces max_parallel. It has no timers or trigger
// logic: mission runs are dispatched explicitly by Command Center.
type MissionConcurrencyTracker struct {
	mu      sync.Mutex
	running map[string]int
	limits  map[string]int
}

func NewMissionConcurrencyTracker(cfg *config.Config) *MissionConcurrencyTracker {
	t := &MissionConcurrencyTracker{running: make(map[string]int), limits: make(map[string]int)}
	t.UpdateConfig(cfg)
	return t
}

func (t *MissionConcurrencyTracker) UpdateConfig(cfg *config.Config) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.limits = make(map[string]int)
	if cfg == nil {
		return
	}
	for _, mission := range cfg.Missions {
		t.limits[mission.Name] = mission.MaxParallel
	}
}

func (t *MissionConcurrencyTracker) NotifyMissionStarted(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	limit := t.limits[name]
	if limit <= 0 {
		limit = 3
	}
	if t.running[name] >= limit {
		return false
	}
	t.running[name]++
	return true
}

func (t *MissionConcurrencyTracker) NotifyMissionDone(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running[name] > 0 {
		t.running[name]--
	}
}
