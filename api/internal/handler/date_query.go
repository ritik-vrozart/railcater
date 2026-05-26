package handler

import (
	"net/http"
	"time"

	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

func defaultAdminPeriod() repository.DateRange {
	now := time.Now().UTC()
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	from := to.AddDate(0, 0, -29)
	return repository.DateRange{From: from, To: to}
}

func parseDateQuery(r *http.Request) (repository.DateRange, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" && toStr == "" {
		return defaultAdminPeriod(), nil
	}
	if fromStr == "" || toStr == "" {
		return repository.DateRange{}, apperror.BadRequest("both from and to are required (YYYY-MM-DD)")
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return repository.DateRange{}, apperror.BadRequest("from must be YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return repository.DateRange{}, apperror.BadRequest("to must be YYYY-MM-DD")
	}
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	if to.Before(from) {
		return repository.DateRange{}, apperror.BadRequest("to must be on or after from")
	}
	return repository.DateRange{From: from, To: to}, nil
}

func orderListFilterFromRequest(r *http.Request, period repository.DateRange) repository.OrderListFilter {
	end := period.EndExclusive()
	f := repository.OrderListFilter{
		Status: r.URL.Query().Get("status"),
		From:   &period.From,
		ToEnd:  &end,
	}
	return f
}
