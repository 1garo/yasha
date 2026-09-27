package account

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) BalanceHandler(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.ErrBadRequest
	}

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
		lg := c.Logger()
		lg.Errorf("failed to get balance: %v", err)
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
			lg := c.Logger()
			lg.Errorf("rows failed: %v", err)
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
		lg := c.Logger()
		lg.Errorf("rows failed: %v", err)
		return echo.ErrInternalServerError
	}
	if !found {
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, map[string]any{"data": balances})
}

func (h *Handler) CreateAccountHandler(c echo.Context) error {
	var request AccountRequest
	if err := c.Bind(&request); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&request); err != nil {
		return err
	}
	id := uuid.New()
	now := time.Now().UTC()
	_, err := h.db.ExecContext(c.Request().Context(),
		`INSERT INTO account (id, first_name, last_name, email, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $5)`,
		id, request.FirstName, request.LastName, request.Email, now,
	)
	if err != nil {
		lg := c.Logger()
		lg.Error("create account", err)
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusCreated, map[string]string{"data": id.String()})
}

func (h *Handler) GetAccountHandler(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.ErrBadRequest
	}
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
		c.Logger().Errorf("failed to query account: %v", err)
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
			c.Logger().Errorf("failed to scan account balance: %v", err)
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
		c.Logger().Errorf("failed to read account balances: %v", err)
		return echo.ErrInternalServerError
	}
	if !found {
		return echo.ErrNotFound
	}
	account.CreatedAt = createdAt.Format(time.RFC3339)
	account.UpdatedAt = updatedAt.Format(time.RFC3339)
	return c.JSON(http.StatusOK, account)
}
