package main

import (
	"net/http"

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

	e.GET("/account", func(c echo.Context) error {
		lg := logger.FromContext(c)
		var response struct {
			Name string `json:"name"`
		}

		response.Name = "Alex"
		lg.Info("successfully retrieved account", zap.String("account_name", response.Name))
		return c.JSON(http.StatusOK, response)
	})

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

	// Example: query cluster name
	var clusterName string
	if err := session.Query(`SELECT cluster_name FROM system.local`).Scan(&clusterName); err != nil {
		e.Logger.Fatal(err)
	}
	e.Logger.Infof("Cluster name: %s", clusterName)

	e.Logger.Fatal(e.Start(":8000"))
}
