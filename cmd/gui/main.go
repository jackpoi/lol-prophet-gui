package main

import (
	"fyne.io/fyne/v2/app"
	gui "github.com/beastars1/lol-prophet-gui"
	"github.com/flopp/go-findfont"
	"os"
	"strings"
)

func init() {
	fontPaths := findfont.List()
	for _, path := range fontPaths {
		if strings.Contains(path, "msyh.ttc") {
			os.Setenv("FYNE_FONT", path)
			break
		}
	}
}

func main() {
	defer os.Unsetenv("FYNE_FONT")
	app := app.New()

	lol := gui.GetLol()
	lol.LoadUI(app)

	go lol.RunProphet()

	app.Run()
}
