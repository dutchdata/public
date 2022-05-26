package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"s3-tool/api"
	"s3-tool/helper"

	"github.com/labstack/echo/v4"
)

func DownloadCSVHandler(c echo.Context) error {
	helper.CSV_path = "output.csv"
	helper.FD = "s3-tool-output"
	headers := []string{"name", "object_count", "total_size_k"}
	helper.WriteCSV(headers, helper.Rows, helper.CSV_path, helper.FD)

	return c.Attachment(helper.Path, "output.csv")
}

func DownloadRecHandler(c echo.Context) error {
	helper.FD = "s3-tool-output"
	helper.REC_path = "recommendation.txt"
	helper.WriteRecommendation(helper.REC_path, helper.FD, helper.Recommendation)

	return c.Attachment(helper.Path, "recommendation.txt")
}

func RecordHandler(c echo.Context) error {
	helper.Rows = api.GetBucketRecords()
	b, _ := json.Marshal(helper.Rows)
	s := string(b)

	n := api.CheckForTrails()
	k := len(n)
	if len(n) > 0 {
		helper.Recommendation = fmt.Sprintf("Found %d CloudTrail(s). More recommendations coming in the next major version :)", k)
		fmt.Println(helper.Recommendation)
	} else {
		helper.Recommendation = "Found 0 CloudTrails. Please enable CloudTrail (including data events) to see more recommendations in the next major version :)"
		fmt.Println(helper.Recommendation)
	}

	return c.String(http.StatusOK, s+helper.Recommendation)
}

func AccessKeyHandler(c echo.Context) error {

	key_id := c.QueryParam("access_key_id")
	key_id_string := url.QueryEscape(key_id)
	os.Setenv("AWS_ACCESS_KEY_ID", key_id_string)

	secret_key := c.QueryParam("secret_key")
	secret_key_string := url.QueryEscape(secret_key)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secret_key_string)

	region := c.QueryParam("region")
	region_string := url.QueryEscape(region)
	os.Setenv("AWS_DEFAULT_REGION", region_string)

	helper.Keys = helper.KeySet{
		Access_key_id: key_id_string,
		Secret_key:    secret_key_string,
		Region:        region_string,
	}

	keys_b, _ := json.Marshal(helper.Keys)
	key_set := string(keys_b)

	return c.String(http.StatusOK, key_set)
}
