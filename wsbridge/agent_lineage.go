package wsbridge

import (
	"slices"
	"sort"

	"squadron/config"
)

type agentLineage struct {
	Missions []agentMissionReference `json:"missions"`
}

type agentMissionReference struct {
	Name string `json:"name"`
}

// buildAgentLineage reports direct configured dependencies, not run history.
// Use the full config: the UI task summary only exposes the first task agent.
func buildAgentLineage(cfg *config.Config, name, missionScope string) agentLineage {
	lineage := agentLineage{Missions: []agentMissionReference{}}
	for _, mission := range cfg.Missions {
		if missionScope != "" {
			if mission.Name != missionScope {
				continue
			}
		} else if mission.GetLocalAgent(name) != nil {
			// A mission-local agent is a separate definition from the global one.
			continue
		}

		usesAgent := slices.Contains(mission.Agents, name)
		for _, task := range mission.Tasks {
			if slices.Contains(task.Agents, name) {
				usesAgent = true
				break
			}
		}
		if usesAgent {
			lineage.Missions = append(lineage.Missions, agentMissionReference{Name: mission.Name})
		}
	}
	sort.Slice(lineage.Missions, func(i, j int) bool {
		return lineage.Missions[i].Name < lineage.Missions[j].Name
	})
	return lineage
}
