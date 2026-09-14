package wsbridge

import (
	"reflect"
	"testing"

	"github.com/mlund01/squadron-wire/protocol"
	"squadron/config"
)

func TestAgentDefinitionLineage(t *testing.T) {
	cfg := &config.Config{
		Agents: []config.Agent{{Name: "reviewer"}, {Name: "unused"}},
		Missions: []config.Mission{
			{Name: "z_pool", Agents: []string{"reviewer"}},
			{Name: "b_task_only", Tasks: []config.Task{{Agents: []string{"other", "reviewer"}}}},
			{Name: "a_repeated", Agents: []string{"reviewer", "reviewer"}, Tasks: []config.Task{
				{Agents: []string{"reviewer"}}, {Agents: []string{"reviewer"}},
			}},
			{Name: "different_agent", Agents: []string{"reviewer_extra"}},
			{Name: "release", Agents: []string{"reviewer"}, LocalAgents: []config.Agent{{Name: "reviewer"}}},
			{Name: "audit", LocalAgents: []config.Agent{{Name: "reviewer"}}, Tasks: []config.Task{{Agents: []string{"reviewer"}}}},
			{Name: "unassigned", LocalAgents: []config.Agent{{Name: "reviewer"}}},
			{Name: "routes_to_release", Tasks: []config.Task{{SendTo: []string{"release"}}}},
		},
	}
	client := NewClient(cfg, true, "", "", nil, "test")
	defer client.Close()
	for _, tc := range []struct {
		name, agent, scope string
		want               []string
	}{
		{"global includes all task agents and deduplicates", "reviewer", "", []string{"a_repeated", "b_task_only", "z_pool"}},
		{"local mission reference", "reviewer", "release", []string{"release"}},
		{"local task reference", "reviewer", "audit", []string{"audit"}},
		{"local declaration alone is not usage", "reviewer", "unassigned", []string{}},
		{"unused global", "unused", "", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := protocol.NewRequest("get_agent_definition", conversationRequest{AgentName: tc.agent, Mission: tc.scope})
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.handleAgentDefinition(request)
			if err != nil {
				t.Fatal(err)
			}
			var definition struct {
				Lineage *agentLineage `json:"lineage"`
			}
			if err := protocol.DecodePayload(response, &definition); err != nil {
				t.Fatal(err)
			}
			if definition.Lineage == nil || definition.Lineage.Missions == nil {
				t.Fatal("lineage must include a non-null missions array, even when unused")
			}
			got := make([]string, 0, len(definition.Lineage.Missions))
			for _, mission := range definition.Lineage.Missions {
				got = append(got, mission.Name)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("missions = %v, want %v", got, tc.want)
			}
		})
	}
}
