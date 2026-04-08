package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Include all files recursively, including paths beginning with '_'
// (SvelteKit outputs frontend assets under /_app/*).
//
//go:embed all:public
var embeddedWeb embed.FS

// RegisterSpaRoutes serves the SvelteKit SPA
func RegisterSpaRoutes(e *echo.Echo) {
	// Extract the "public" subdirectory from the embedded filesystem
	webFS, err := fs.Sub(embeddedWeb, "public")
	if err != nil {
		panic(err)
	}

	// Serve all static assets
	e.GET("/*", echo.WrapHandler(http.FileServer(http.FS(webFS))))
}
