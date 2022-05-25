package helper

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"s3-tool/api"

	echo "github.com/labstack/echo/v4"
)

var (
	CSV_path       string
	REC_path       string
	Recommendation string
	FD             string
	Path           string
	Rows           [][]string
	Keys           KeySet
)

type KeySet struct {
	Access_key_id string `json:"access_key_id"`
	Secret_key    string `json:"secret_key"`
	Region        string `json:"region"`
}

func DownloadHandler(c echo.Context) error {
	CSV_path = "output.csv"
	FD = "s3-tool-output"
	headers := []string{"name", "object_count", "total_size_k"}
	WriteCSV(headers, Rows, CSV_path, FD)

	REC_path = "recommendation.txt"
	WriteRecommendation(REC_path, FD, Recommendation)

	return c.Attachment(Path, "output.csv")
}

func RecordHandler(c echo.Context) error {
	Rows = api.GetBucketRecords()
	b, _ := json.Marshal(Rows)
	s := string(b)

	n := api.CheckForTrails()
	k := len(n)
	if len(n) > 0 {
		Recommendation = fmt.Sprintf("Found %d CloudTrail(s). More recommendations coming in the next major version :)", k)
		fmt.Println(Recommendation)
	} else {
		Recommendation = "Found 0 CloudTrails. Please enable CloudTrail (including data events) to see more recommendations in the next major version :)"
		fmt.Println(Recommendation)
	}

	return c.String(http.StatusOK, s+Recommendation)
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

	Keys = KeySet{
		Access_key_id: key_id_string,
		Secret_key:    secret_key_string,
		Region:        region_string,
	}

	keys_b, _ := json.Marshal(Keys)
	key_set := string(keys_b)

	return c.String(http.StatusOK, key_set)
}

func WriteCSV(headers []string, rows [][]string, file string, directory string) (output_file *os.File) {
	output_file, Path = PathResolver(file, directory)
	defer output_file.Close()
	csv_writer := NewCSVWriter(output_file, headers)
	for i := range rows {
		csv_writer.Write(rows[i])
	}
	csv_writer.Flush()
	return output_file
}

func NewCSVWriter(file *os.File, headers []string) (writer *csv.Writer) {
	writer = csv.NewWriter(file)
	writer.Write(headers)
	return writer
}

func WriteRecommendation(file string, directory string, recommendation string) (output_file *os.File) {
	output_file, Path = PathResolver(file, directory)
	defer output_file.Close()
	rec_writer := NewRecommendationWriter(output_file)
	rec_writer.WriteString(recommendation)
	rec_writer.Flush()
	return output_file
}

func NewRecommendationWriter(file *os.File) (writer *bufio.Writer) {
	writer = bufio.NewWriter(file)
	return writer
}

func PathResolver(target_file_name string, parent_directory string) (file *os.File, Path string) {
	root_directory, _ := os.UserHomeDir()
	os.Mkdir(root_directory+"/"+parent_directory, 0755)
	Path = root_directory + "/" + parent_directory + "/" + target_file_name
	file, _ = os.Create(Path)
	return file, Path
}
