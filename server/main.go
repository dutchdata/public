package main

import (
	"net/http"
	"s3-tool/api"
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
	e.GET("/get", helper.DownloadHandler)
	e.GET("/check-trails", helper.TrailCheckHandler)

	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api.GetBucketRecords()

	// e.Logger.Fatal(e.Start(":8080"))
}
