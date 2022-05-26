package main

import (
	"net/http"
	"s3-tool/helper"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/auth", helper.AccessKeyHandler)
	e.GET("/go", helper.RecordHandler)
	e.GET("/getcsv", helper.DownloadCSVHandler)
	e.GET("/getrec", helper.DownloadRecHandler)

	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Logger.Fatal(e.Start(":8080"))
}
