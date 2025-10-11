package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/1garo/yasha/logger"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AccountRequest struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Currency  string `json:"currency" validate:"required"`
}

type Account struct {
	ID        gocql.UUID `json:"id"`
	FirstName *string    `json:"firstName"`
	LastName  *string    `json:"lastName"`
	Email     *string    `json:"email"`
	Currency  *string    `json:"currency"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updateAt,omitempty"`
	Balance   int        `json:"balance"` // TODO: default to zero now, change on next steps
}

type TxType string

const (
	TxTypeDeposit  TxType = "deposit"
	TxTypeWithdraw TxType = "withdraw"
	TxTypeTransfer TxType = "transfer"
)

func (tx TxType) IsValid() bool {
	switch tx {
	case TxTypeDeposit, TxTypeWithdraw, TxTypeTransfer:
		return true
	default:
		return false
	}
}

type TransactionRequest struct {
	AccountId   string `json:"accountId" validate:"required"`
	Type        TxType `json:"type" validate:"required"`
	AmountMinor int    `json:"amountMinor" validate:"required"`
	Currency    string `json:"currency" validate:"required"`
	To          string `json:"to,omitempty"`
}

type BalanceResponse struct {
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency,omitempty"`
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.ErrUnprocessableEntity
	}
	return nil
}

func RequestBodyLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			l := logger.FromContext(c)
			req := c.Request()
			if req.Body == nil || req.Method == http.MethodGet {
				return next(c)
			}

			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				l.Error("failed to read body", zap.Error(err))
				return next(c)
			}

			// Restore the body so Echo can read it
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Try to unmarshal JSON
			var bodyMap map[string]any
			if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
				// fallback: log raw string if not JSON
				l.Info("not JSON", zap.String("body", string(bodyBytes)))
			} else {
				l.Info("request body", zap.Any("body", bodyMap))
			}

			return next(c)
		}
	}
}

func availableCurrencies() map[string]struct{} {
	return map[string]struct{}{
		"GBP": {},
		"BRL": {},
		"USD": {},
		"EUR": {},
	}
}

func directionFromTxType(txType TxType) string {
	if txType == TxTypeDeposit {
		return "credit"
	}

	return "debit"
}

func insertIntoLedger(db *gocql.Session, tx *TransactionRequest, lg *zap.Logger, direction string, description string, txnId, entryId gocql.UUID) error {
	createdAt := time.Now().UTC()

	lg = lg.With(
		zap.String("direction", direction),
		zap.String("accountId", tx.AccountId),
		zap.String("txID", txnId.String()),
		zap.String("entryId", entryId.String()),
		zap.Time("createdAt", createdAt),
	)

	query := `insert into ledger_entries (account_id, entry_id, transaction_id, direction, amount_minor, currency, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	lg.Info("insert new entry into ledger", zap.String("query", query))

	err := db.Query(query,
		tx.AccountId,
		entryId,
		txnId,
		direction,
		tx.AmountMinor,
		tx.Currency,
		description,
		createdAt,
	).Exec()
	if err != nil {
		lg.Error("failed to insert new entry to ledger", zap.Error(err))
		return echo.ErrInternalServerError
	}

	return nil
}

func main() {
	e := echo.New()

	l := logger.NewCtxLogger()

	e.Logger.Info("Starting cluster")
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Port = 9042
	cluster.Keyspace = "yasha"
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer func(session *gocql.Session) {
		if session != nil {
			session.Close()
		}
	}(session)

	e.Logger.Info("Connected to Cassandra")

	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(RequestBodyLogger())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("db", session)
			return next(c)
		}
	})
	e.Use(middleware.Recover())
	e.Use(middleware.RequestIDWithConfig(
		middleware.RequestIDConfig{
			Generator: func() string {
				return uuid.New().String()
			},
		}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogRequestID: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.FromContext(c).Sugar().Infof("finished: %s", v.URI)
			return nil
		},
	}))
	e.Use(l.LoggerMiddleware())
	e.Logger.SetLevel(log.INFO)

	e.GET("/account/:id/balance", func(c echo.Context) error {
		lg := logger.FromContext(c)
		db := c.Get("db").(*gocql.Session)

		id := c.Param("id")
		lg = lg.With(
			zap.String("account_id", id),
		)

		query := `select direction, amount_minor, currency from ledger_entries where account_id = ?`
		iter := db.Query(query, id).Iter()

		var (
			direction string
			curr      string
			amount    int64
		)

		lg.Info("getting the direction and amount from ledger entries")
		amountMinor := int64(0)
		currency := ""
		for iter.Scan(&direction, &amount, &curr) {
			if currency == "" {
				currency = curr
			}

			switch direction {
			case "debit":
				amountMinor -= amount
			case "credit":
				amountMinor += amount
			}
		}

		if err := iter.Close(); err != nil {
			lg.Error("failed to retrieve account balance", zap.Error(err))
			return echo.ErrInternalServerError
		}

		lg.Info("successfully retrieved the balance account")
		return c.JSON(http.StatusOK, map[string]any{
			"data": BalanceResponse{
				AmountMinor: amountMinor,
				Currency:    currency,
			},
		})
	})

	e.POST("/transaction", func(c echo.Context) error {
		lg := logger.FromContext(c)
		db := c.Get("db").(*gocql.Session)

		tx := new(TransactionRequest)
		if err = c.Bind(tx); err != nil {
			return echo.ErrBadRequest
		}

		if err = c.Validate(tx); err != nil {
			return err
		}

		if _, ok := availableCurrencies()[tx.Currency]; !ok {
			lg.Error("unsupported currency", zap.String("currency", tx.Currency))
			return echo.NewHTTPError(http.StatusBadRequest, "Unsupported currency")
		}

		if !tx.Type.IsValid() {
			lg.Error("invalid transaction type", zap.String("type", string(tx.Type)))
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction type")
		}

		txId, _ := gocql.RandomUUID()
		if tx.Type != TxTypeTransfer {
			direction := directionFromTxType(tx.Type)
			entryId := gocql.TimeUUID()
			if err := insertIntoLedger(db, tx, lg, direction, string(tx.Type), txId, entryId); err != nil {
				return err
			}
		} else if tx.To != "" {
			// TODO: check if who is being debited have enough balance
			createdAt := time.Now().UTC()

			debitID := gocql.TimeUUID()
			direction := directionFromTxType(TxTypeWithdraw)
			description := fmt.Sprintf("Transfer to %s", tx.To)
			if err := insertIntoLedger(db, tx, lg, direction, description, txId, debitID); err != nil {
				return err
			}

			if err = session.Query(`
				  INSERT INTO ledger_entries_by_transaction (transaction_id, account_id, entry_id, direction, amount_minor, currency, description, created_at)
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				txId, tx.AccountId, debitID, direction, tx.AmountMinor, tx.Currency, "Transfer to Bob", createdAt,
			).Exec(); err != nil {
				lg.Error("failed to transaction into ledger entries")
				return echo.ErrInternalServerError
			}

			description = fmt.Sprintf("Received from %s", tx.AccountId)
			tx.AccountId = tx.To
			creditID := gocql.TimeUUID()
			direction = directionFromTxType(TxTypeDeposit)
			if err = insertIntoLedger(db, tx, lg, direction, description, txId, creditID); err != nil {
				return err
			}
			if err = session.Query(`
				  INSERT INTO ledger_entries_by_transaction (transaction_id, account_id, entry_id, direction, amount_minor, currency, description, created_at)
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				txId, tx.AccountId, creditID, direction, tx.AmountMinor, tx.Currency, "Received from Alice", createdAt,
			).Exec(); err != nil {
				lg.Error("failed to transaction into ledger entries")
				return echo.ErrInternalServerError
			}
		}

		lg.Info("successfully created the transaction entry in the ledger")
		return c.JSON(http.StatusCreated, map[string]any{
			"data": map[string]string{"transactionId": txId.String()},
		})
	})
	e.POST("/account", func(c echo.Context) error {
		lg := logger.FromContext(c)
		db := c.Get("db").(*gocql.Session)

		acc := new(AccountRequest)
		if err = c.Bind(acc); err != nil {
			return echo.ErrBadRequest
		}

		if err = c.Validate(acc); err != nil {
			return err
		}

		if _, ok := availableCurrencies()[acc.Currency]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Unsupported currency")
		}

		id, _ := gocql.RandomUUID()
		lg = lg.With(zap.Any("account_id", id))

		q := `insert into account(id, first_name, last_name, email, currency, created_at) values (?, ?, ?, ?, ?, ?)`
		lg.Info("inserting new account into db", zap.String("query", q))
		if err := db.Query(q,
			id,
			acc.FirstName,
			acc.LastName,
			acc.Email,
			acc.Currency,
			time.Now().UTC(),
		).Exec(); err != nil {
			lg.Error("failed to insert new account", zap.Error(err))
			return echo.ErrInternalServerError
		}

		lg.Info("successfully created a new account")
		return c.JSON(http.StatusCreated, map[string]string{
			"data": id.String(),
		})
	})

	e.GET("/account/:id", func(c echo.Context) error {
		lg := logger.FromContext(c)
		db := c.Get("db").(*gocql.Session)

		id, err := gocql.ParseUUID(c.Param("id"))
		if err != nil {
			return echo.ErrBadRequest
		}

		var (
			acc       Account
			createdAt *time.Time
			updatedAt *time.Time
		)

		lg = lg.With(zap.Any("account_id", id))

		query := `SELECT id, first_name, last_name, email, currency, created_at, updated_at FROM account WHERE id = ? LIMIT 1`
		lg.Info("querying account from db", zap.String("query", query))

		if err = db.Query(query, id).Scan(
			&acc.ID,
			&acc.FirstName,
			&acc.LastName,
			&acc.Email,
			&acc.Currency,
			&createdAt,
			&updatedAt,
		); err != nil {
			lg.Error("failed to query account", zap.Error(err))
			if errors.Is(err, gocql.ErrNotFound) {
				return echo.ErrNotFound
			}

			return echo.ErrInternalServerError
		}

		acc.CreatedAt = createdAt.Format(time.RFC3339)
		if updatedAt != nil {
			acc.UpdatedAt = updatedAt.Format(time.RFC3339)
		}

		lg.Info("successfully retrieved account")
		return c.JSON(http.StatusOK, acc)
	})

	e.Logger.Fatal(e.Start(":8000"))
}
