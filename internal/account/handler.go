package account

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/1garo/yasha/internal/currency"
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
	var (
		balance  int64
		currency string
	)
	if err = h.db.QueryRowContext(
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
		acc.currency as currency
		FROM account as acc
		LEFT JOIN ledger_entries as le ON le.account_id = acc.id
		WHERE acc.id = $1
		GROUP BY acc.id, acc.currency`, id,
	).Scan(
		&balance,
		&currency,
	); err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": BalanceResponse{AmountMinor: balance, Currency: currency}})
}

func (h *Handler) CreateAccountHandler(c echo.Context) error {
	var request AccountRequest
	if err := c.Bind(&request); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&request); err != nil {
		return err
	}
	if _, ok := currency.AvailableCurrencies()[request.Currency]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Unsupported currency")
	}

	id := uuid.New()
	now := time.Now().UTC()
	_, err := h.db.ExecContext(c.Request().Context(),
		`INSERT INTO account (id, first_name, last_name, email, currency, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		id, request.FirstName, request.LastName, request.Email, request.Currency, now,
	)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"data": id.String()})
}

func (h *Handler) GetAccountHandler(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.ErrBadRequest
	}
	var (
		account              Account
		createdAt, updatedAt time.Time
	)
	err = h.db.QueryRowContext(
		c.Request().Context(),
		`SELECT id, first_name, last_name, email, currency, created_at, updated_at FROM account WHERE id = $1`, id).
		Scan(&account.ID, &account.FirstName, &account.LastName, &account.Email, &account.Currency, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return echo.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	account.CreatedAt = createdAt.Format(time.RFC3339)
	account.UpdatedAt = updatedAt.Format(time.RFC3339)
	return c.JSON(http.StatusOK, account)
}
