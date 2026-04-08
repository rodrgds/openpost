package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// Include all files recursively, including paths beginning with '_'
// (SvelteKit outputs frontend assets under /_app/*).
//
//go:embed all:public
var embeddedWeb embed.FS

// RegisterSpaRoutes serves the SvelteKit SPA
func RegisterSpaRoutes(e *echo.Echo) {
	webFS, err := fs.Sub(embeddedWeb, "public")
	if err != nil {
		panic(err)
	}

	staticHandler := http.FileServer(http.FS(webFS))

	e.GET("/*", func(c echo.Context) error {
		reqPath := c.Request().URL.Path

		if strings.HasPrefix(reqPath, "/api") {
			return echo.NewHTTPError(http.StatusNotFound, "API not found")
		}

		staticHandler.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	})

	e.GET("/", func(c echo.Context) error {
		return serveIndex(webFS, c)
	})
}

func serveIndex(webFS fs.FS, c echo.Context) error {
	indexData, err := fs.ReadFile(webFS, "index.html")
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "index.html not found")
	}

	c.Response().Header().Set("Content-Type", "text/html")
	c.Response().Write(indexData)
	return nil
}
