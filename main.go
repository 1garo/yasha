package main

import (
	"github.com/1garo/yasha/internal/config"
	"github.com/1garo/yasha/internal/server"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	srv, err := server.New(e, config.Load())
	if err != nil {
		e.Logger.Fatal(err)
	}
	srv.InitMiddleware()
	srv.RegisterRoutes()

	e.Logger.Fatal(e.Start(":8000"))
}

//func dbFrom(c echo.Context) *sql.DB { return c.Get("db").(*sql.DB) }
