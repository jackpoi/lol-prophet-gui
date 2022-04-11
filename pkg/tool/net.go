package tool

import (
	"github.com/avast/retry-go"
	"github.com/beastars1/lol-prophet-gui/services/logger"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

func HttpGet(url string) []byte {
	var body []byte
	err := retry.Do(func() error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return err
	}, retry.Delay(time.Millisecond*10), retry.Attempts(5))
	if err != nil {
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelError)
			scope.SetExtra("url", url)
			scope.SetExtra("error", err.Error())
			scope.SetExtra("errorVerbose", errors.Errorf("%+v", err))
			sentry.CaptureMessage("http请求失败")
		})
		logger.Debug("http请求失败", zap.Error(err), url)
		return nil
	}
	return body
}
