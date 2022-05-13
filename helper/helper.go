package helper

import (
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
	Target_file_name string
	Target_file_directory string
	Output_file *os.File
	Path string
	Rows [][]string
	Keys KeySet
)

type KeySet struct {
	Access_key_id string `json:"access_key_id"`
	Secret_key string `json:"secret_key"`
	Region string `json:"region"`
}

func TrailCheckHandler(c echo.Context) error {
	api.CheckForTrails()
	return c.String(http.StatusOK,"check log")
}

func TrailEventHandler(c echo.Context) error {
	api.CheckForEvents()
	return c.String(http.StatusOK,"check log")
}

func DownloadHandler(c echo.Context) error {
	Target_file_name = "output.csv"
	Target_file_directory = "s3-tool-output"
	headers := []string{"name","object_count","total_size_k"}
	WriteRecords(headers,Rows,Target_file_name,Target_file_directory)
	fmt.Println("File downloaded to",Path)
	return c.Attachment(Path,"output.csv")
}

func RecordHandler(c echo.Context) error {
	Rows = api.GetBucketRecords()
	b, _ := json.Marshal(Rows)
	s := string(b)
	return c.String(http.StatusOK,s)
}

func AccessKeyHandler(c echo.Context) (error) {

	key_id := c.QueryParam("access_key_id")
	key_id_string := url.QueryEscape(key_id)
	os.Setenv("AWS_ACCESS_KEY_ID",key_id_string)

	secret_key := c.QueryParam("secret_key") 
	secret_key_string := url.QueryEscape(secret_key)
	os.Setenv("AWS_SECRET_ACCESS_KEY",secret_key_string)

	region := c.QueryParam("region") 
	region_string := url.QueryEscape(region)
	os.Setenv("AWS_DEFAULT_REGION",region_string)

	Keys = KeySet{
		Access_key_id: key_id_string,
		Secret_key: secret_key_string,
		Region: region_string,
	}

	keys_b, _ := json.Marshal(Keys)
	key_set := string(keys_b)
	
	return c.String(http.StatusOK, key_set)
}

func WriteRecords(headers []string,rows [][]string,file string, directory string) (output_file *os.File) {
	output_file, Path = PathResolver(file,directory)
	defer output_file.Close()
	writer := NewRecordWriter(output_file,headers)
	for i := range rows {
		writer.Write(rows[i])
	}
	writer.Flush()
	return output_file
}

func NewRecordWriter(file *os.File,headers []string) (writer *csv.Writer) {
	writer = csv.NewWriter(file)
	writer.Write(headers)
	return writer
}

func PathResolver(target_file_name string, parent_directory string) (file *os.File, Path string) {
	root_directory, _ := os.UserHomeDir()
	os.Mkdir(root_directory + "/" + parent_directory,0755)
	Path = root_directory + "/" + parent_directory + "/" + target_file_name
	file, _ = os.Create(Path)
	return file, Path
}




