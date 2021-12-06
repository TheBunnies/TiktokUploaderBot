package tiktok

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type DownloadModel struct {
	Status      string `json:"status"`
	DownloadUrl string `json:"download_url"`
	Message     string `json:"message"`
}

func GetDownloadModel(url string) (DownloadModel, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.tiktokdownloader.org/check.php?v=%s", url))
	if err != nil {
		return DownloadModel{}, errors.New("cannot get the download link")
	}
	defer resp.Body.Close()
	model := DownloadModel{}
	json.NewDecoder(resp.Body).Decode(&model)
	return model, nil
}
func (model *DownloadModel) DownloadVideo() (string, error) {
	resp, err := http.Get(model.DownloadUrl)
	if err != nil {
		return "", errors.New("cannot download the video")
	}
	defer resp.Body.Close()
	filename := model.GetFilename()
	dir, _ := os.Getwd()
	filepath := filepath.Join(dir, filename)
	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return filename, nil
}
func (model *DownloadModel) GetFilename() string {
	filename := strings.Split(model.DownloadUrl, "=")[1] + ".mp4"
	return filename
}
func (model *DownloadModel) GetVideoAsReader() (io.Reader, error) {
	resp, err := http.Get(model.DownloadUrl)
	if err != nil {
		return resp.Body, err
	}
	return resp.Body, nil
}
func (model *DownloadModel) GetConverted() (*os.File, error) {
	filename, err := model.DownloadVideo()
	if err != nil {
		return nil, err
	}
	dir, _ := os.Getwd()
	newFileName := filepath.Join(dir, "temp_"+filename)
	err = ffmpeg_go.Input(filepath.Join(dir, filename)).
		Output(newFileName, ffmpeg_go.KwArgs{"c:v": "libx264"}).
		OverWriteOutput().Run()
	if err != nil {
		fmt.Printf("Filename: %s \n New filename: %s", filename, newFileName)
		return nil, err
	}
	file, err := os.Open(newFileName)
	if err != nil {
		return nil, err
	}
	os.Remove(filename)
	return file, nil
}
