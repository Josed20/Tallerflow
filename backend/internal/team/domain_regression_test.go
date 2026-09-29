package team

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConsumeResultUsesFrontendContractFieldNames(t *testing.T) {
	// Regression: ISSUE-002 — invitation success returned Email/Role while the frontend reads email/role.
	// Found by /qa on 2026-09-29.
	// Report: .gstack/qa-reports/qa-report-localhost-2026-09-29.md
	payload, err := json.Marshal(ConsumeResult{Email: "member@example.test", Role: "OPERATOR"})
	if err != nil {
		t.Fatal(err)
	}

	got := string(payload)
	if !strings.Contains(got, `"email":"member@example.test"`) || !strings.Contains(got, `"role":"OPERATOR"`) {
		t.Fatalf("consume result JSON = %s, want lowercase API field names", got)
	}
	if strings.Contains(got, `"Email"`) || strings.Contains(got, `"Role"`) {
		t.Fatalf("consume result JSON exposed Go field names: %s", got)
	}
}
