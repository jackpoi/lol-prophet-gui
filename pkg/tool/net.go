package tool

import (
	"github.com/beastars1/lol-prophet-gui/services/logger"
	"io/ioutil"
	"net/http"
)

func HttpGet(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("http get error", err)
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("io read error", err)
		return nil
	}
	return body
}
