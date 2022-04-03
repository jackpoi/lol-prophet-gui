package lol_prophet_gui

import "github.com/beastars1/lol-prophet-gui/global"

var (
	APPVersion = "0.2.3"
	Commit     = "dev"
	BuildTime  = ""
	BuildUser  = ""
)

func init() {
	global.SetAppInfo(global.AppInfo{
		Version:   APPVersion,
		Commit:    Commit,
		BuildUser: BuildUser,
		BuildTime: BuildTime,
	})
}
