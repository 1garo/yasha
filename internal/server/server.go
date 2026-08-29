package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/1garo/yasha/internal/config"
	"github.com/1garo/yasha/logger"
	"github.com/1garo/yasha/validator"
	v "github.com/go-playground/validator"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap"
)

type server struct {
	e       *echo.Echo
	log     *logger.CtxLogger
	session *gocql.Session
}

func NewServer(e *echo.Echo, cfg config.Config) (server, error) {
	log := logger.NewCtxLogger()
	session, err := initSession(cfg)
	if err != nil {
		return server{}, err
	}

	return server{
		e,
		log,
		session,
	}, nil
}

func (s *server) InitServer() {

}

func initSession(cfg config.Config) (*gocql.Session, error) {
	fmt.Println("Starting cluster")
	port, err := strconv.Atoi(cfg.CassandraPort)
	if err != nil {
		return nil, err
	}

	cluster := gocql.NewCluster(cfg.CassandraHost)
	cluster.Port = port
	cluster.Keyspace = cfg.CassandraKeyspace
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to Cassandra")

	return session, nil
}

func (s *server) InitMiddleware() {
	s.e.Logger.SetLevel(log.INFO)
	s.e.Validator = &validator.CustomValidator{Validator: v.New()}
	s.e.Use(RequestBodyLogger())
	s.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("db", s.session)
			return next(c)
		}
	})
	s.e.Use(middleware.Recover())
	s.e.Use(middleware.RequestIDWithConfig(
		middleware.RequestIDConfig{
			Generator: func() string {
				return uuid.New().String()
			},
		}))
	s.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogRequestID: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.FromContext(c).Sugar().Infof("finished: %s", v.URI)
			return nil
		},
	}))
	s.e.Use(s.log.LoggerMiddleware())
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
