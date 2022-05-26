package helper

import (
	"bufio"
	"encoding/csv"
	"os"
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
