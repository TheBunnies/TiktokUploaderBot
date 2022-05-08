package tiktok

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const Origin = "http://api2.musical.ly"

type AwemeDetail struct {
	Author struct {
		Unique_ID string
	}
	Aweme_ID    string
	Create_Time int64
	Desc        string
	Video       struct {
		Duration  int64
		Play_Addr struct {
			Width    int
			Height   int
			URL_List []string
		}
	}
}

func Parse(id string) (uint64, error) {
	return strconv.ParseUint(id, 10, 64)
}

func NewAwemeDetail(id uint64, transport http.RoundTripper) (*AwemeDetail, error) {
	req, err := http.NewRequest("GET", Origin+"/aweme/v1/aweme/detail/", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = "aweme_id=" + strconv.FormatUint(id, 10)
	client := &http.Client{
		Transport: transport,
	}
	res, err := client.Do(req)
	if err != nil {
		client.Transport = nil
		res, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}
	var detail struct {
		Aweme_Detail AwemeDetail
	}
	if err := json.NewDecoder(res.Body).Decode(&detail); err != nil {
		return nil, err
	}
	return &detail.Aweme_Detail, nil
}

func (a AwemeDetail) Duration() time.Duration {
	return time.Duration(a.Video.Duration) * time.Millisecond
}
func (a AwemeDetail) Description() string {
	return strings.TrimSpace(a.Desc)
}

func (a AwemeDetail) Time() string {
	return strings.Replace(time.Unix(a.Create_Time, 0).Format("Mon, 02 Jan 2006 15:04:05 MST"), "  ", " ", -1)
}

func (a AwemeDetail) URL() (string, error) {
	if len(a.Video.Play_Addr.URL_List) == 0 {
		return "", errors.New("invalid slice")
	}
	first := a.Video.Play_Addr.URL_List[0]
	loc, err := url.Parse(first)
	if err != nil {
		return "", err
	}
	loc.RawQuery = ""
	loc.Scheme = "http"
	return loc.String(), nil
}

func GetId(uri string) (string, error) {
	url, _ := url.Parse(uri)
	url.RawQuery = ""
	url.Scheme = "http"
	resp, err := http.Get(url.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("video not found")
	}
	newUrl, _ := url.Parse(resp.Request.URL.String())
	newUrl.RawQuery = ""
	newUrl.Scheme = "http"
	fmt.Println(newUrl.String())
	return fileNameWithoutExtension(filepath.Base(newUrl.String())), nil
}

func fileNameWithoutExtension(fileName string) string {
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}

func (det *AwemeDetail) DownloadVideo(downloadBytesLimit int64) (*os.File, error) {
	addr, err := det.URL()
	if err != nil {
		return nil, err
	}
	res, err := http.Get(addr)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	size, _ := strconv.Atoi(res.Header.Get("Content-Length"))
	downloadSize := int64(size)
	if downloadSize > downloadBytesLimit {
		return nil, errors.New("download file is too large, upgrade your server premium level to be able to upload larger videos")
	}
	u, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s.%s", u.String(), strings.Split(res.Header.Get("Content-Type"), "/")[1])
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if _, err := file.ReadFrom(res.Body); err != nil {
		return nil, err
	}
	openedFile, err := os.Open(file.Name())
	if err != nil {
		log.Println(err)
		openedFile.Close()
		os.Remove(openedFile.Name())
		return nil, err
	}
	return openedFile, nil
}
