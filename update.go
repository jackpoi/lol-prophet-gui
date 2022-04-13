package lol_prophet_gui

import (
	"encoding/json"
	"github.com/beastars1/lol-prophet-gui/global"
	"github.com/beastars1/lol-prophet-gui/pkg/tool"
	"time"
)

const (
	releaseInfoUrl = "https://api.github.com/repos/beastars1/lol-prophet-gui/releases/latest"
	releaseUrl     = "https://github.com/beastars1/lol-prophet-gui/releases"
)

type releaseInfo struct {
	HtmlUrl         string    `json:"html_url"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []struct {
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadUrl string    `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"`
}

func (r releaseInfo) hasNewVersion() (ok bool, downloadUrl, info string) {
	currVersion := global.Version
	latestVersion := r.TagName[1:]
	if currVersion < latestVersion {
		// 有新版本
		return true, r.Assets[0].BrowserDownloadUrl, r.Body
	}
	return false, "", ""
}

func CheckUpdate() (ok bool, downloadUrl, info string) {
	body := tool.HttpGet(releaseInfoUrl)
	var releaseInfo = releaseInfo{}
	err := json.Unmarshal(body, &releaseInfo)
	if err != nil {
		return false, "", ""
	}
	if ok, downloadUrl, info := releaseInfo.hasNewVersion(); ok {
		return ok, downloadUrl, info
	}
	return false, "", ""
}
