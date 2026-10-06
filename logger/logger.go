package logger

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ctxLoggerKey struct{}

type CtxLogger struct {
	logger *zap.Logger
}

func NewCtxLogger() (*CtxLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &CtxLogger{logger: logger}, nil
}

func FromContext(c echo.Context) *zap.Logger {
	if l, ok := c.Request().Context().Value(ctxLoggerKey{}).(*zap.Logger); ok {
		return l
	}

	return zap.NewNop()
}

func (cl *CtxLogger) LoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestLogger := cl.logger.With(
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.String("request_id", c.Response().Header().Get(echo.HeaderXRequestID)),
			)

			ctx := context.WithValue(c.Request().Context(), ctxLoggerKey{}, requestLogger)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
