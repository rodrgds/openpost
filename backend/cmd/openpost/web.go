package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

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

		htmlFallback := path + ".html"
		if _, err := fs.Stat(webFS, htmlFallback); err == nil {
			data, _ := fs.ReadFile(webFS, htmlFallback)
			c.Response().Header().Set("Content-Type", "text/html")
			c.Response().Write(data)
			return nil
		}

		info, err := fs.Stat(webFS, path)
		if err == nil {
			if info.IsDir() {
				indexPath := path + "/index.html"
				if _, err := fs.Stat(webFS, indexPath); err == nil {
					indexData, _ := fs.ReadFile(webFS, indexPath)
					c.Response().Header().Set("Content-Type", "text/html")
					c.Response().Write(indexData)
					return nil
				}
				return echo.NewHTTPError(http.StatusNotFound, "directory index not found")
			}

			file, _ := webFS.Open(path)
			defer file.Close()
			http.ServeContent(c.Response().Writer, c.Request(), info.Name(), info.ModTime(), file.(http.File))
			return nil
		}

		if os.IsNotExist(err) {
			indexData, _ := fs.ReadFile(webFS, "index.html")
			c.Response().Header().Set("Content-Type", "text/html")
			c.Response().Write(indexData)
			return nil
		}

		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	})

	e.GET("/", func(c echo.Context) error {
		indexData, _ := fs.ReadFile(webFS, "index.html")
		c.Response().Header().Set("Content-Type", "text/html")
		c.Response().Write(indexData)
		return nil
	})
}
