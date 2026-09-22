package server

import (
	"database/sql"

	"github.com/1garo/yasha/internal/config"
	"github.com/1garo/yasha/internal/db"
	"github.com/1garo/yasha/logger"
	"github.com/1garo/yasha/validator"
	v "github.com/go-playground/validator"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type server struct {
	e   *echo.Echo
	log *logger.CtxLogger
	db  *sql.DB
}

func InitServer(e *echo.Echo, cfg config.Config) (server, error) {
	log := logger.NewCtxLogger()
	db, err := db.InitDB(cfg)
	if err != nil {
		return server{}, err
	}

	return server{
		e,
		log,
		db,
	}, nil
}

func (s *server) InitMiddleware() {
	s.e.Logger.SetLevel(log.INFO)
	s.e.Validator = &validator.CustomValidator{Validator: v.New()}
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
