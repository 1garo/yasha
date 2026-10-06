package transaction

import (
	"context"
	"database/sql"
	"errors"
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
	lg := logger.FromContext(c).With(zap.String("operation", "create_transaction"))
	var request TransactionRequest
	if err := c.Bind(&request); err != nil {
		lg.Warn("failed to bind request", zap.Error(err))
		return echo.ErrBadRequest
	}
	if err := c.Validate(&request); err != nil {
		lg.Warn("request validation failed", zap.Error(err))
		return err
	}
	if _, ok := currency.AvailableCurrencies()[request.Currency]; !ok {
		lg.Warn("unsupported currency", zap.String("currency", request.Currency))
		return echo.NewHTTPError(http.StatusBadRequest, "Unsupported currency")
	}
	if !request.Type.IsValid() || request.AmountMinor <= 0 {
		lg.Warn("invalid transaction request",
			zap.String("transaction_type", string(request.Type)),
			zap.Int64("amount_minor", request.AmountMinor),
		)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction")
	}
	accountID, err := uuid.Parse(request.AccountID)
	if err != nil {
		lg.Warn("invalid source account id", zap.Error(err))
		return echo.ErrBadRequest
	}

	transactionID := uuid.New()
	lg = lg.With(
		zap.String("account_id", accountID.String()),
		zap.String("transaction_type", string(request.Type)),
		zap.String("currency", request.Currency),
		zap.Int64("amount_minor", request.AmountMinor),
		zap.String("transaction_id", transactionID.String()),
	)
	now := time.Now().UTC()
	tx, err := h.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		lg.Error("failed to begin transaction", zap.Error(err))
		return echo.ErrInternalServerError
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			lg.Error("failed to roll back transaction", zap.Error(rollbackErr))
		}
	}()

	switch request.Type {
	case TxTypeTransfer:
		toID, parseErr := uuid.Parse(request.To)
		if parseErr != nil {
			lg.Warn("invalid transfer destination", zap.Error(parseErr))
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination account")
		}
		lg = lg.With(zap.String("destination_account_id", toID.String()))
		if toID == accountID {
			lg.Warn("transfer source and destination are the same account")
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination account")
		}
		if err = insertLedger(c.Request().Context(), tx, accountID, transactionID, "debit", request, "Transfer", now); err == nil {
			err = insertLedger(c.Request().Context(), tx, toID, transactionID, "credit", request, "Transfer", now)
		}
	case TxTypeDeposit, TxTypeWithdraw:
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
		lg.Error("failed to commit", zap.Error(err))
		return echo.ErrInternalServerError
	}
	lg.Info("transaction created")
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"transactionId": transactionID.String()}})
}

func insertLedger(ctx context.Context, tx *sql.Tx, accountID, transactionID uuid.UUID, direction string, request TransactionRequest, description string, createdAt time.Time) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO ledger_entries (account_id, entry_id, transaction_id, direction, amount_minor, currency, description, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		accountID, uuid.New(), transactionID, direction, request.AmountMinor, request.Currency, description, createdAt)
	return err
}
