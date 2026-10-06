package account

import (
	"database/sql"
	"net/http"
	"time"

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

func (h *Handler) BalanceHandler(c echo.Context) error {
	lg := logger.FromContext(c).With(zap.String("operation", "get_balance"))
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		lg.Warn("invalid account id", zap.Error(err))
		return echo.ErrBadRequest
	}
	lg = lg.With(zap.String("account_id", id.String()))

	rows, err := h.db.QueryContext(
		c.Request().Context(),
		`SELECT COALESCE(
			SUM(
				CASE 
				WHEN direction = 'credit' 
				THEN amount_minor ELSE -amount_minor 
				END
			), 
			0
		) AS balance,
		le.currency as currency
		FROM account AS acc
		LEFT JOIN ledger_entries AS le ON le.account_id = acc.id
		WHERE acc.id = $1
		GROUP BY acc.id, le.currency`, id,
	)
	if err != nil {
		lg.Error("failed to get balance", zap.Error(err))
		return echo.ErrInternalServerError
	}
	defer rows.Close()

	balances := make([]BalanceResponse, 0)
	found := false
	for rows.Next() {
		var (
			balance  int64
			currency sql.NullString
		)
		if err := rows.Scan(&balance, &currency); err != nil {
			lg.Error("failed to scan rows", zap.Error(err))
			return echo.ErrInternalServerError
		}

		found = true
		if currency.Valid {
			balances = append(balances, BalanceResponse{
				AmountMinor: balance,
				Currency:    currency.String,
			})
		}
	}

	if err := rows.Err(); err != nil {
		lg.Error("failed while reading account balances", zap.Error(err))
		return echo.ErrInternalServerError
	}
	if !found {
		lg.Debug("account not found")
		return echo.ErrNotFound
	}

	lg.Info("balance retrieved", zap.Int("balance_count", len(balances)))
	return c.JSON(http.StatusOK, map[string]any{"data": balances})
}

func (h *Handler) CreateAccountHandler(c echo.Context) error {
	lg := logger.FromContext(c).With(zap.String("operation", "create_account"))
	var request AccountRequest
	if err := c.Bind(&request); err != nil {
		lg.Warn("failed to bind request", zap.Error(err))
		return echo.ErrBadRequest
	}
	if err := c.Validate(&request); err != nil {
		lg.Warn("request validation failed", zap.Error(err))
		return err
	}

	id := uuid.New()
	lg = lg.With(zap.String("account_id", id.String()))
	now := time.Now().UTC()
	_, err := h.db.ExecContext(c.Request().Context(),
		`INSERT INTO account (id, first_name, last_name, email, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $5)`,
		id, request.FirstName, request.LastName, request.Email, now,
	)
	if err != nil {
		lg.Error("failed to create account in database", zap.Error(err))
		return echo.ErrInternalServerError
	}
	lg.Info("account created")
	return c.JSON(http.StatusCreated, map[string]string{"data": id.String()})
}

func (h *Handler) GetAccountHandler(c echo.Context) error {
	lg := logger.FromContext(c).With(zap.String("operation", "get_account"))
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		lg.Warn("invalid account id", zap.Error(err))
		return echo.ErrBadRequest
	}
	lg = lg.With(zap.String("account_id", id.String()))
	rows, err := h.db.QueryContext(
		c.Request().Context(),
		`SELECT 
		acc.id, 
		acc.first_name, 
		acc.last_name, 
		acc.email, 
		acc.created_at, 
		acc.updated_at,
		COALESCE(
			SUM(
				CASE
				WHEN le.direction = 'credit' 
				THEN amount_minor
				ELSE -amount_minor
				END
		), 0) AS balance,
		le.currency
		FROM account as acc
		LEFT JOIN ledger_entries as le ON le.account_id = acc.id
		WHERE acc.id = $1
		GROUP BY acc.id, le.currency`, id)
	if err != nil {
		lg.Error("failed to query account", zap.Error(err))
		return echo.ErrInternalServerError
	}
	defer rows.Close()

	var (
		account              Account
		createdAt, updatedAt time.Time
		found                bool
	)
	for rows.Next() {
		var (
			balance  int64
			currency sql.NullString
		)
		if err := rows.Scan(
			&account.ID,
			&account.FirstName,
			&account.LastName,
			&account.Email,
			&createdAt,
			&updatedAt,
			&balance,
			&currency,
		); err != nil {
			lg.Error("failed to scan account balance", zap.Error(err))
			return echo.ErrInternalServerError
		}
		found = true
		if currency.Valid {
			account.Balances = append(account.Balances, BalanceResponse{
				AmountMinor: balance,
				Currency:    currency.String,
			})
		}
	}
	if err := rows.Err(); err != nil {
		lg.Error("failed to read account balances", zap.Error(err))
		return echo.ErrInternalServerError
	}
	if !found {
		lg.Debug("account not found")
		return echo.ErrNotFound
	}
	account.CreatedAt = createdAt.Format(time.RFC3339)
	account.UpdatedAt = updatedAt.Format(time.RFC3339)
	lg.Info("account retrieved", zap.Int("balance_count", len(account.Balances)))
	return c.JSON(http.StatusOK, account)
}
