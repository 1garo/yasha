package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/1garo/yasha/internal/currency"
	"github.com/1garo/yasha/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) TransactionHandler(c echo.Context) error {
	lg := logger.FromContext(c)
	var request TransactionRequest
	if err := c.Bind(&request); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&request); err != nil {
		return err
	}
	if _, ok := currency.AvailableCurrencies()[request.Currency]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Unsupported currency")
	}
	if !request.Type.IsValid() || request.AmountMinor <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction")
	}
	accountID, err := uuid.Parse(request.AccountID)
	if err != nil {
		return echo.ErrBadRequest
	}
	transactionID := uuid.New()
	now := time.Now().UTC()
	tx, err := h.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	if request.Type == TxTypeTransfer {
		toID, parseErr := uuid.Parse(request.To)
		if parseErr != nil || toID == accountID {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination account")
		}
		if err = insertLedger(c.Request().Context(), tx, accountID, transactionID, "debit", request, "Transfer", now); err == nil {
			err = insertLedger(c.Request().Context(), tx, toID, transactionID, "credit", request, "Transfer", now)
		}
	} else {
		direction := "debit"
		if request.Type == TxTypeDeposit {
			direction = "credit"
		}
		err = insertLedger(c.Request().Context(), tx, accountID, transactionID, direction, request, string(request.Type), now)
	}
	if err != nil {
		lg.Error("failed to write ledger", zap.Error(err))
		return echo.ErrInternalServerError
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"transactionId": transactionID.String()}})
}

func insertLedger(ctx context.Context, tx *sql.Tx, accountID, transactionID uuid.UUID, direction string, request TransactionRequest, description string, createdAt time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries (account_id, entry_id, transaction_id, direction, amount_minor, currency, description, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, accountID, uuid.New(), transactionID, direction, request.AmountMinor, request.Currency, description, createdAt)
	return err
}
