package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// Include all files recursively, including paths beginning with '_'
// (SvelteKit outputs frontend assets under /_app/*).
//
//go:embed all:public
var embeddedWeb embed.FS

func RegisterSpaRoutes(e *echo.Echo) {
	webFS, err := fs.Sub(embeddedWeb, "public")
	if err != nil {
		panic(err)
	}

	e.GET("/*", func(c echo.Context) error {
		reqPath := c.Request().URL.Path

		if strings.HasPrefix(reqPath, "/api") {
			return echo.NewHTTPError(http.StatusNotFound, "API not found")
		}

		path := strings.TrimPrefix(reqPath, "/")

		info, err := fs.Stat(webFS, path)
		if err == nil {
			if info.IsDir() {
				indexPath := path
				if indexPath != "" {
					indexPath = indexPath + "/"
				}
				indexPath = indexPath + "index.html"

				if _, err := fs.Stat(webFS, indexPath); err == nil {
					indexData, err := fs.ReadFile(webFS, indexPath)
					if err == nil {
						c.Response().Header().Set("Content-Type", "text/html")
						c.Response().Write(indexData)
						return nil
					}
				}

				return echo.NewHTTPError(http.StatusNotFound, "directory index not found")
			}

			file, err := webFS.Open(path)
			if err != nil {
				return echo.NewHTTPError(http.StatusNotFound, "file not found")
			}
			defer file.Close()

			http.ServeContent(c.Response().Writer, c.Request(), info.Name(), info.ModTime(), file.(http.File))
			return nil
		}

		if os.IsNotExist(err) {
			indexData, err := fs.ReadFile(webFS, "index.html")
			if err != nil {
				return echo.NewHTTPError(http.StatusNotFound, "index.html not found")
			}
			c.Response().Header().Set("Content-Type", "text/html")
			c.Response().Write(indexData)
			return nil
		}

		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	})

	e.GET("/", func(c echo.Context) error {
		indexData, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "index.html not found")
		}
		c.Response().Header().Set("Content-Type", "text/html")
		c.Response().Write(indexData)
		return nil
	})
}
