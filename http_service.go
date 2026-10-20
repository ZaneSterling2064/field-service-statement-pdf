package statement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Service struct {
	infrai *InfraiClient
	logger *slog.Logger
	now    func() time.Time
}

func NewService(infrai *InfraiClient, logger *slog.Logger) *Service {
	return &Service{infrai: infrai, logger: logger, now: time.Now}
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /statements", s.exportStatement)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func (s *Service) exportStatement(w http.ResponseWriter, r *http.Request) {
	var input StatementRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if input.AccountName == "" || input.PeriodLabel == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account_name and period_label are required"})
		return
	}

	usage, err := s.infrai.UsageTimeseries(r.Context())
	if err != nil {
		s.writeUpstreamError(w, err)
		return
	}
	statement := BuildStatement(input, usage, s.now())
	html, err := RenderHTML(statement)
	if err != nil {
		s.logger.Error("statement render failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "statement rendering failed"})
		return
	}

	digest := sha256.Sum256([]byte(input.AccountName + "\x00" + input.PeriodLabel))
	pdf, err := s.infrai.GeneratePDF(r.Context(), html, "statement-"+hex.EncodeToString(digest[:16]))
	if err != nil {
		s.writeUpstreamError(w, err)
		return
	}
	s.logger.Info("statement exported", "account", input.AccountName, "work_orders", len(statement.WorkOrders), "total", statement.Total)
	writeJSON(w, http.StatusOK, map[string]any{
		"included_work_orders": len(statement.WorkOrders),
		"statement_total":      statement.Total,
		"pdf":                  json.RawMessage(pdf),
	})
}

func (s *Service) writeUpstreamError(w http.ResponseWriter, err error) {
	var apiErr *InfraiError
	if errors.As(err, &apiErr) {
		status := apiErr.HTTPStatus
		if status < 400 || status >= 500 {
			status = http.StatusBadGateway
		}
		writeJSON(w, status, map[string]string{"error": apiErr.Code, "message": apiErr.Message})
		return
	}
	s.logger.Error("upstream request failed", "error", err)
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
