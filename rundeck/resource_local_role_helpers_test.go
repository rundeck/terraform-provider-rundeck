package rundeck

import "testing"

// These are plain unit tests for the pure helper functions
// resource_local_role_framework.go relies on to resolve usernames to IDs and
// determine role membership - unlike the resource's CRUD methods, they need
// no live server to exercise.

func TestRoleIDFromResponse(t *testing.T) {
	got, err := roleIDFromResponse(map[string]interface{}{"id": float64(42)})
	if err != nil {
		t.Fatalf("roleIDFromResponse: %v", err)
	}
	if got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

func TestRoleIDFromResponse_missingID(t *testing.T) {
	if _, err := roleIDFromResponse(map[string]interface{}{"authority": "foo"}); err == nil {
		t.Error("want an error when the response has no id field")
	}
}

func TestRoleIDFromResponse_wrongType(t *testing.T) {
	if _, err := roleIDFromResponse(map[string]interface{}{"id": "42"}); err == nil {
		t.Error("want an error when id is not a float64 (e.g. a JSON string, not a number)")
	}
}

func TestUsernameToIDMap(t *testing.T) {
	users := []map[string]interface{}{
		{"username": "alice", "id": float64(1)},
		{"username": "bob", "id": float64(2)},
		// Missing/wrong-typed fields are skipped rather than panicking.
		{"username": "carol"},
		{"id": float64(3)},
	}

	got := usernameToIDMap(users)

	want := map[string]int64{"alice": 1, "bob": 2}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %v", len(got), len(want), got)
	}
	for username, id := range want {
		if got[username] != id {
			t.Errorf("got[%q] = %d, want %d", username, got[username], id)
		}
	}
}

func TestMembersOfRole(t *testing.T) {
	users := []map[string]interface{}{
		{
			"username": "alice",
			"roles": []interface{}{
				map[string]interface{}{"id": float64(10)},
				map[string]interface{}{"id": float64(20)},
			},
		},
		{
			"username": "bob",
			"roles": []interface{}{
				map[string]interface{}{"id": float64(20)},
			},
		},
		{
			"username": "carol",
			"roles":    []interface{}{},
		},
	}

	got := membersOfRole(users, 20)

	want := []string{"alice", "bob"} // sorted
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestMembersOfRole_noneMatch(t *testing.T) {
	users := []map[string]interface{}{
		{"username": "alice", "roles": []interface{}{map[string]interface{}{"id": float64(10)}}},
	}

	got := membersOfRole(users, 999)

	// Must be a non-nil empty slice, not nil: converting a nil slice to a
	// Terraform Set produces null rather than an empty known set, which is
	// the same "inconsistent result after apply" bug class this function's
	// own doc comment calls out.
	if got == nil {
		t.Fatal("got nil, want a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}
