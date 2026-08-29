package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type propertyRecord struct {
	ID             string    `json:"id"`
	MaintenanceDue time.Time `json:"maintenance_due"`
	DocumentDue    time.Time `json:"document_due"`
	InspectionDue  time.Time `json:"inspection_due"`
}

type cleanupDecision struct {
	RecordID string   `json:"record_id"`
	Reasons  []string `json:"reasons"`
}

func staleReasons(record propertyRecord, now time.Time) []string {
	var reasons []string
	if record.MaintenanceDue.Before(now) {
		reasons = append(reasons, "maintenance_request")
	}
	if record.DocumentDue.Before(now) {
		reasons = append(reasons, "tenant_document")
	}
	if record.InspectionDue.Before(now) {
		reasons = append(reasons, "inspection_reminder")
	}
	return reasons
}

func decideCleanup(record propertyRecord, now time.Time) (cleanupDecision, bool) {
	reasons := staleReasons(record, now)
	if len(reasons) == 0 {
		return cleanupDecision{}, false
	}
	return cleanupDecision{RecordID: record.ID, Reasons: reasons}, true
}

func main() {
	client, err := newClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	now := time.Now().UTC()
	record := propertyRecord{
		ID:             "unit-204",
		MaintenanceDue: now.Add(-48 * time.Hour),
		DocumentDue:    now.Add(24 * time.Hour),
		InspectionDue:  now.Add(-72 * time.Hour),
	}
	decision, stale := decideCleanup(record, now)
	if !stale {
		fmt.Println("no stale property records")
		return
	}
	payload, err := json.Marshal(decision)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := client.publish(string(payload), "property-sweep-"+record.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	jobID, err := client.createCron("0 2 * * *", os.Getenv("PROPERTY_SWEEP_TASK_URL"), "property-sweep-schedule")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("queued cleanup for %s and scheduled job %s\n", decision.RecordID, jobID)
}
