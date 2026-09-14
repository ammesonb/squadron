package wsbridge

import (
	"testing"

	"squadron/config"
)

func TestMissionConcurrencyTracker(t *testing.T) {
	tracker := NewMissionConcurrencyTracker(&config.Config{Missions: []config.Mission{{Name: "one", MaxParallel: 1}}})
	if !tracker.NotifyMissionStarted("one") {
		t.Fatal("first run should be accepted")
	}
	if tracker.NotifyMissionStarted("one") {
		t.Fatal("second run should be rejected at capacity")
	}
	tracker.NotifyMissionDone("one")
	if !tracker.NotifyMissionStarted("one") {
		t.Fatal("run should be accepted after completion")
	}
}

func TestMissionConcurrencyTrackerReloadsLimits(t *testing.T) {
	tracker := NewMissionConcurrencyTracker(&config.Config{Missions: []config.Mission{{Name: "mission", MaxParallel: 1}}})
	if !tracker.NotifyMissionStarted("mission") {
		t.Fatal("first run should be accepted")
	}
	tracker.UpdateConfig(&config.Config{Missions: []config.Mission{{Name: "mission", MaxParallel: 2}}})
	if !tracker.NotifyMissionStarted("mission") {
		t.Fatal("updated limit should be applied")
	}
}
