package main

import (
	"fyne.io/fyne/v2/app"
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

	lol := newLol()
	lol.loadUI(app)

	app.Run()
}
