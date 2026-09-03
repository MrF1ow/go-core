package log

import (
	"os"
	"strings"
	"testing"
)

func TestActivityLogSQL_AllQueriesAcceptAppID(t *testing.T) {
	sql, err := os.ReadFile("queries/activity_log.sql")
	if err != nil {
		sql, err = os.ReadFile("../queries/activity_log.sql")
	}
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	for _, name := range []string{"CountAllActivityLogs", "ListAllActivityLogs", "ExportAllActivityLogs"} {
		if !strings.Contains(text, "-- name: "+name) {
			t.Fatalf("missing %s", name)
		}
	}
	want := "sqlc.narg('app_id')::uuid"
	if strings.Count(text, want) < 3 {
		t.Fatalf("expected narg app_id on count, list, and export, got %d", strings.Count(text, want))
	}
}
