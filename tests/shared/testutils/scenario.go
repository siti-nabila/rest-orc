package testutils

import "testing"

type Scenario struct {
	Name string
	Run  func(*testing.T)
}

func RunScenarios(t *testing.T, scenarios []Scenario) {
	t.Helper()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, scenario.Run)
	}
}
