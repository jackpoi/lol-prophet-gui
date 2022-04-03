package bootstrap

import (
	"io"
	"net/http"
	"testing"
)

func TestIp(t *testing.T) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", "https://api.ip.sb/ip", nil)

	request.Header.Set("user-agent", "lol")

	resp, err := client.Do(request)
	if err != nil {
		t.Error(err)
		return
	}
	bts, _ := io.ReadAll(resp.Body)
	t.Log(string(bts))
}
