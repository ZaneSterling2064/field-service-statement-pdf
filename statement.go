package statement

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// WorkOrder is the field record considered for an account statement.
type WorkOrder struct {
	ID             string             `json:"id"`
	Site           string             `json:"site"`
	DispatchStatus string             `json:"dispatch_status"`
	MeteringStatus string             `json:"metering_status"`
	CompletedAt    time.Time          `json:"completed_at"`
	Amount         float64            `json:"amount"`
	Photos         []Photo            `json:"photos"`
	FollowUp       TechnicianFollowUp `json:"technician_follow_up"`
}

type Photo struct {
	Source  string `json:"source"`
	Caption string `json:"caption"`
}

type TechnicianFollowUp struct {
	Technician string     `json:"technician"`
	Summary    string     `json:"summary"`
	DueAt      *time.Time `json:"due_at,omitempty"`
}

type StatementRequest struct {
	AccountName string      `json:"account_name"`
	PeriodLabel string      `json:"period_label"`
	WorkOrders  []WorkOrder `json:"work_orders"`
}

type Statement struct {
	AccountName string
	PeriodLabel string
	GeneratedAt string
	WorkOrders  []WorkOrder
	Total       float64
	UsageJSON   string
}

// BuildStatement applies the billing boundary: completed and settled work only.
func BuildStatement(in StatementRequest, usageJSON []byte, now time.Time) Statement {
	statement := Statement{
		AccountName: in.AccountName,
		PeriodLabel: in.PeriodLabel,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		UsageJSON:   string(usageJSON),
	}
	for _, order := range in.WorkOrders {
		if order.DispatchStatus != "completed" || order.MeteringStatus != "settled" {
			continue
		}
		statement.WorkOrders = append(statement.WorkOrders, order)
		statement.Total += order.Amount
	}
	return statement
}

var statementTemplate = template.Must(template.New("statement").Funcs(template.FuncMap{
	"money": func(value float64) string { return fmt.Sprintf("%.2f", value) },
	"date":  func(value time.Time) string { return value.UTC().Format("2006-01-02 15:04 UTC") },
	"join":  strings.Join,
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><style>
body{font-family:Arial,sans-serif;color:#182028;margin:36px}h1{margin-bottom:4px}.meta{color:#52606d}
table{width:100%;border-collapse:collapse;margin-top:24px}th,td{text-align:left;border-bottom:1px solid #d9e2ec;padding:9px;vertical-align:top}
.photos{display:flex;gap:8px;flex-wrap:wrap}.photos figure{margin:4px}.photos img{width:140px;height:96px;object-fit:cover}.usage{white-space:pre-wrap;font:11px monospace;background:#f3f5f7;padding:12px}
</style></head><body>
<h1>Account statement</h1><div class="meta">{{.AccountName}} · {{.PeriodLabel}} · generated {{.GeneratedAt}}</div>
<table><thead><tr><th>Work order</th><th>Dispatch</th><th>Technician follow-up</th><th>Amount</th></tr></thead><tbody>
{{range .WorkOrders}}<tr><td><strong>{{.ID}}</strong><br>{{.Site}}<br>{{date .CompletedAt}}
{{if .Photos}}<div class="photos">{{range .Photos}}<figure><img src="{{.Source}}" alt="{{.Caption}}"><figcaption>{{.Caption}}</figcaption></figure>{{end}}</div>{{end}}
</td><td>{{.DispatchStatus}}</td><td><strong>{{.FollowUp.Technician}}</strong><br>{{.FollowUp.Summary}}{{if .FollowUp.DueAt}}<br>Due {{date .FollowUp.DueAt}}{{end}}</td><td>{{money .Amount}}</td></tr>{{end}}
</tbody><tfoot><tr><th colspan="3">Total</th><th>{{money .Total}}</th></tr></tfoot></table>
<h2>Metered account usage</h2><div class="usage">{{.UsageJSON}}</div>
</body></html>`))

func RenderHTML(statement Statement) (string, error) {
	var out bytes.Buffer
	if err := statementTemplate.Execute(&out, statement); err != nil {
		return "", fmt.Errorf("render statement: %w", err)
	}
	return out.String(), nil
}
