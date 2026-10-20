package statement

import (
	"testing"
	"time"
)

func TestBuildStatementBillingBoundary(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		dispatch  string
		metering  string
		wantN     int
		wantTotal float64
	}{
		{name: "completed and settled is billed", dispatch: "completed", metering: "settled", wantN: 1, wantTotal: 125.50},
		{name: "active dispatch is deferred", dispatch: "en_route", metering: "settled", wantN: 0, wantTotal: 0},
		{name: "unsettled meter is deferred", dispatch: "completed", metering: "pending", wantN: 0, wantTotal: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildStatement(StatementRequest{
				AccountName: "North Plant",
				PeriodLabel: "September 2026",
				WorkOrders:  []WorkOrder{{ID: "WO-1042", DispatchStatus: tt.dispatch, MeteringStatus: tt.metering, Amount: 125.50}},
			}, []byte(`{"units":18}`), now)
			if len(got.WorkOrders) != tt.wantN {
				t.Fatalf("included work orders = %d, want %d", len(got.WorkOrders), tt.wantN)
			}
			if got.Total != tt.wantTotal {
				t.Fatalf("total = %.2f, want %.2f", got.Total, tt.wantTotal)
			}
		})
	}
}
