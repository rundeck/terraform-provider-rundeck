package rundeck

import (
	"testing"

	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

// flattenRunnerProjectAssociations previously ranged over a map to build its
// result, making the list order nondeterministic between reads - a noisy
// diff for a data source even when nothing about the associations changed.
func TestFlattenRunnerProjectAssociations_sortedByProjectName(t *testing.T) {
	nodeFilters := map[string]string{
		"zebra": "tag:z",
		"alpha": "tag:a",
		"mike":  "tag:m",
	}
	assoc := &openapi.RunnerProjectAssociations{
		ProjectNodeFilters: &nodeFilters,
	}

	result := flattenRunnerProjectAssociations(assoc)

	if len(result) != 3 {
		t.Fatalf("got %d entries, want 3", len(result))
	}
	want := []string{"alpha", "mike", "zebra"}
	for i, name := range want {
		if got := result[i].ProjectName.ValueString(); got != name {
			t.Errorf("result[%d].ProjectName = %q, want %q (result not sorted: %v)", i, got, name, result)
		}
	}
}

func TestFlattenRunnerProjectAssociations_nilWhenEmpty(t *testing.T) {
	if got := flattenRunnerProjectAssociations(nil); got != nil {
		t.Errorf("got %v, want nil for a nil association", got)
	}
	if got := flattenRunnerProjectAssociations(&openapi.RunnerProjectAssociations{}); got != nil {
		t.Errorf("got %v, want nil when no project has any association", got)
	}
}
