package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
}

type Account struct {
	ID        gocql.UUID `json:"id"`
	FirstName *string    `json:"firstName"`
	LastName  *string    `json:"lastName"`
	Email     *string    `json:"email"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updateAt"`
	Balance   int        `json:"balance"` // TODO: default to zero now, change on next steps
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
            if req.Body == nil || req.Method == http.MethodGet{
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

		id := gocql.TimeUUID()
		q := `insert into account(id, first_name, last_name, email, created_at) values (?, ?, ?, ?, ?)`
		lg = lg.With(
			// TODO: I don't know if I want the query to show up in all info logs, maybe just once before the query
			zap.String("query", q), zap.Any("account_id", id),
		)
		if err := db.Query(q,
			id,
			acc.FirstName,
			acc.LastName,
			acc.Email,
			gocql.TimeUUID().Time(),
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

		var acc Account
		query := `SELECT id, first_name, last_name, email, created_at, updated_at FROM account WHERE id = ? LIMIT 1`
		lg = lg.With(
			// TODO: I don't know if I want the query to show up in all info logs, maybe just once before the query
			zap.String("query", query), zap.Any("account_id", id),
		)

		if err = db.Query(query, id).Scan(
			&acc.ID,
			&acc.FirstName,
			&acc.LastName,
			&acc.Email,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		); err != nil {
			lg.Error("failed to query account", zap.Error(err))
			if errors.Is(err, gocql.ErrNotFound) {
				return echo.ErrNotFound
			}

			return echo.ErrInternalServerError
		}

		lg.Info("successfully retrieved account")
		return c.JSON(http.StatusOK, acc)
	})

	e.Logger.Fatal(e.Start(":8000"))
}
