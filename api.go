package lol_prophet_gui

import (
	"github.com/beastars1/lol-prophet-gui/conf"
	"github.com/beastars1/lol-prophet-gui/global"
)

func getAll() *conf.Client {
	return &global.DefaultClientConf
}
