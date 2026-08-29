package main

import (
	"testing"
	"time"
)

func TestDecideCleanupNamesOnlyStalePropertyWork(t *testing.T) {
	now := time.Date(2026, 8, 10, 2, 0, 0, 0, time.UTC)
	record := propertyRecord{
		ID:             "unit-204",
		MaintenanceDue: now.Add(-time.Hour),
		DocumentDue:    now.Add(time.Hour),
		InspectionDue:  now.Add(-2 * time.Hour),
	}
	decision, stale := decideCleanup(record, now)
	if !stale || decision.RecordID != "unit-204" {
		t.Fatalf("expected stale unit-204, got %#v, stale=%v", decision, stale)
	}
	want := []string{"maintenance_request", "inspection_reminder"}
	if len(decision.Reasons) != len(want) {
		t.Fatalf("expected %v, got %v", want, decision.Reasons)
	}
	for i := range want {
		if decision.Reasons[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, decision.Reasons)
		}
	}
}

func TestDecideCleanupLeavesCurrentRecordAlone(t *testing.T) {
	now := time.Date(2026, 8, 10, 2, 0, 0, 0, time.UTC)
	record := propertyRecord{ID: "unit-205", MaintenanceDue: now, DocumentDue: now.Add(time.Hour), InspectionDue: now.Add(2 * time.Hour)}
	if _, stale := decideCleanup(record, now); stale {
		t.Fatal("current property record should stay out of the cleanup queue")
	}
}
