package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/1garo/yasha/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	e := echo.New()

	l := logger.NewCtxLogger()

	e.Logger.Info("Starting cluster")
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Port = 9042
	cluster.Keyspace = "system"
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

	e.GET("/account/:id", func(c echo.Context) error {
		lg := logger.FromContext(c)
		db := c.Get("db").(*gocql.Session)

		type Account struct {
			ID        gocql.UUID
			FirstName string
			LastName  string
			Email     string
			CreatedAt time.Time
			UpdatedAt time.Time
		}
		acc := Account{}
		id, _ := strconv.Atoi(c.QueryParam("id"))
		iter := db.Query(
			`SELECT id, first_name, last_name, email, created_at, updated_at 
			 FROM account WHERE id = ? LIMIT 1`, id,
		).Iter()
		defer iter.Close()
		if rows := iter.Scan(&acc.ID, &acc.FirstName, &acc.LastName, &acc.Email, &acc.CreatedAt, &acc.UpdatedAt); !rows {
			return echo.ErrNotFound
		}

		lg.Info("successfully retrieved account", zap.Any("account_id", acc.ID))
		return c.JSON(http.StatusOK, acc)
	})

	e.Logger.Fatal(e.Start(":8000"))
}
