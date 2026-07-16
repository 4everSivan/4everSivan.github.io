package report

import (
	"strings"
	"testing"
	"time"

	"github.com/4everSivan/4everSivan.github.io/internal/approval"
	"github.com/4everSivan/4everSivan.github.io/internal/scanner"
)

func TestInventoryClassifiesEveryCompletedResultWithoutReasonText(t *testing.T) {
	t.Parallel()
	result := scanner.Result{
		RelativePath: "安全/示例.md",
		SHA256:       strings.Repeat("a", 64),
		Completed:    true,
		Findings: []scanner.Finding{{
			RuleID: scanner.RuleLocalResource, Level: scanner.LevelBlock,
			RelativePath: "安全/示例.md", Line: 3, Reason: "must-never-be-persisted", Approvable: false,
		}},
	}
	inventory, err := InventoryFromResults([]scanner.Result{result}, time.Date(2026, 7, 15, 1, 2, 3, 0, time.UTC), approval.New())
	if err != nil {
		t.Fatal(err)
	}
	if inventory.CandidateCount != 1 || inventory.PassedCount != 0 || inventory.ExcludedCount != 1 {
		t.Fatalf("unexpected counts: %+v", inventory)
	}
	if inventory.Documents[0].Status != InventoryExcluded || inventory.Documents[0].Findings[0].RuleID != scanner.RuleLocalResource {
		t.Fatalf("unexpected classification: %+v", inventory.Documents[0])
	}
}
