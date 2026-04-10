package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
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
		if reqPath == "" {
			reqPath = "/"
		}

		if strings.HasPrefix(reqPath, "/api") {
			return echo.NewHTTPError(http.StatusNotFound, "API not found")
		}

		relPath := strings.TrimPrefix(path.Clean(reqPath), "/")
		if relPath == "." {
			relPath = ""
		}

		if relPath == "" {
			indexData, _ := fs.ReadFile(webFS, "index.html")
			c.Response().Header().Set("Content-Type", "text/html")
			c.Response().Write(indexData)
			return nil
		}

		htmlFile := relPath + ".html"
		if _, err := fs.Stat(webFS, htmlFile); err == nil {
			data, _ := fs.ReadFile(webFS, htmlFile)
			c.Response().Header().Set("Content-Type", "text/html")
			c.Response().Write(data)
			return nil
		}

		info, err := fs.Stat(webFS, relPath)
		if err == nil {
			if info.IsDir() {
				indexPath := relPath + "/index.html"
				if _, err := fs.Stat(webFS, indexPath); err == nil {
					indexData, _ := fs.ReadFile(webFS, indexPath)
					c.Response().Header().Set("Content-Type", "text/html")
					c.Response().Write(indexData)
					return nil
				}

				indexData, _ := fs.ReadFile(webFS, "index.html")
				c.Response().Header().Set("Content-Type", "text/html")
				c.Response().Write(indexData)
				return nil
			}

			hfs := http.FS(webFS)
			file, _ := hfs.Open(relPath)
			defer file.Close()
			http.ServeContent(c.Response().Writer, c.Request(), info.Name(), info.ModTime(), file)
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
}
