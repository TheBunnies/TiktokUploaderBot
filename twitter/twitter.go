package twitter

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/gocolly/colly"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type VideoDownloader struct {
	VideoUrl    string
	BearerToken string
	GuestToken  string
}

var (
	rgxBearer  = regexp.MustCompile(`"Bearer.*?"`)
	rgxNum     = regexp.MustCompile(`[0-9]+`)
	rgxAddress = regexp.MustCompile(`https.*m3u8`)
	rgxFormat  = regexp.MustCompile(`.*m3u8`)
)

func NewTwitterVideoDownloader(url string) *VideoDownloader {
	self := new(VideoDownloader)
	self.VideoUrl = url
	return self
}

func (s *VideoDownloader) GetBearerToken(proxy string) string {
	c := colly.NewCollector()
	err := c.SetProxy(proxy)
	if err != nil {
		log.Println(err)
	}

	c.OnResponse(func(r *colly.Response) {
		s.BearerToken = strings.Trim(rgxBearer.FindString(string(r.Body)), `"`)
	})

	c.Visit("https://abs.twimg.com/web-video-player/TwitterVideoPlayerIframe.cefd459559024bfb.js")

	return s.BearerToken
}

func (s *VideoDownloader) GetXGuestToken(proxy string) string {
	c := colly.NewCollector()
	c.SetProxy(proxy)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Authorization", s.BearerToken)
	})

	c.OnResponse(func(r *colly.Response) {
		s.GuestToken = rgxNum.FindString(string(r.Body))
	})

	c.Post("https://api.twitter.com/1.1/guest/activate.json", nil)

	return s.GuestToken
}

func (s *VideoDownloader) GetM3U8Urls(proxy string) string {
	var m3u8_urls string

	c := colly.NewCollector()
	c.SetProxy(proxy)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Authorization", s.BearerToken)
		r.Headers.Set("x-guest-token", s.GuestToken)
	})

	c.OnResponse(func(r *colly.Response) {
		m3u8_urls = strings.ReplaceAll(rgxAddress.FindString(string(r.Body)), "\\", "")
	})

	url := "https://api.twitter.com/1.1/videos/tweet/config/" +
		strings.TrimPrefix(s.VideoUrl, "https://twitter.com/i/status/") +
		".json"

	c.Visit(url)

	return m3u8_urls
}

func (s *VideoDownloader) GetM3U8Url(m3u8_urls string, proxy string) string {
	var m3u8_url string

	c := colly.NewCollector()
	c.SetProxy(proxy)

	c.OnResponse(func(r *colly.Response) {
		m3u8_urls := rgxFormat.FindAllString(string(r.Body), -1)
		m3u8_url = "https://video.twimg.com" + m3u8_urls[len(m3u8_urls)-1]
	})

	c.Visit(m3u8_urls)

	return m3u8_url
}

func (s *VideoDownloader) Download(proxy string, downloadBytesLimit int64) (*os.File, error) {
	s.GetBearerToken(proxy)
	s.GetXGuestToken(proxy)
	m3u8_urls := s.GetM3U8Urls(proxy)
	m3u8_url := s.GetM3U8Url(m3u8_urls, proxy)

	sum := md5.Sum([]byte(m3u8_url))
	filename := hex.EncodeToString(sum[:]) + ".mp4"
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 60 * time.Second,
	}
	c := &http.Client{
		Transport: t,
	}

	response, err := c.Get(m3u8_url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	size, _ := strconv.Atoi(response.Header.Get("Content-Length"))
	downloadSize := int64(size)
	if downloadSize > downloadBytesLimit {
		return nil, errors.New("download file is too large, upgrade your server premium level to be able to upload larger videos")
	}
	cmd := exec.Command("ffmpeg", "-y", "-i", m3u8_url, "-c", "copy", filename)
	cmd.Run()
	openedFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return openedFile, nil
}
