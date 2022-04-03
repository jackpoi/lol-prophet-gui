package lol_prophet_gui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/beastars1/lol-prophet-gui/bootstrap"
	"github.com/beastars1/lol-prophet-gui/conf"
	"github.com/beastars1/lol-prophet-gui/global"
	"sync"
	"time"
)

const (
	layout = "2006-01-02 15:01:05"
)

var (
	lol  *gui
	once sync.Once
)

type gui struct {
	output *widget.Entry
	conf   *conf.Client
	window fyne.Window
	p      *Prophet
}

func (g *gui) RunProphet() {
	g.p.Run()
}

func (g *gui) LoadUI(app fyne.App) {
	g.output = widget.NewMultiLineEntry()
	g.output.TextStyle.Monospace = false

	w := app.NewWindow(global.AppName)

	checkConf := container.NewGridWithColumns(4,
		widget.NewCheckWithData("自动接受对局", binding.BindBool(&g.conf.AutoAcceptGame)),
		widget.NewCheckWithData("选择英雄界面自动发送", binding.BindBool(&g.conf.AutoSendTeamHorse)),
		widget.NewCheckWithData("发送自己马匹信息", binding.BindBool(&g.conf.ShouldSendSelfHorse)),
		container.NewHBox(
			widget.NewLabel("选择英雄"),
			widget.NewEntryWithData(binding.IntToString(binding.BindInt(&g.conf.ChooseChampSendMsgDelaySec))),
			widget.NewLabel("秒后发送"),
		),
	)

	horse0Name := binding.BindString(&g.conf.HorseNameConf[0])
	horse1Name := binding.BindString(&g.conf.HorseNameConf[1])
	horse2Name := binding.BindString(&g.conf.HorseNameConf[2])
	horse3Name := binding.BindString(&g.conf.HorseNameConf[3])
	horse4Name := binding.BindString(&g.conf.HorseNameConf[4])
	horse5Name := binding.BindString(&g.conf.HorseNameConf[5])
	horseConf := container.NewGridWithColumns(6,
		widget.NewEntryWithData(horse0Name),
		widget.NewEntryWithData(horse1Name),
		widget.NewEntryWithData(horse2Name),
		widget.NewEntryWithData(horse3Name),
		widget.NewEntryWithData(horse4Name),
		widget.NewEntryWithData(horse5Name),
	)

	horseCheck := container.NewGridWithColumns(6,
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[0])), widget.NewLabelWithData(horse0Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[1])), widget.NewLabelWithData(horse1Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[2])), widget.NewLabelWithData(horse2Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[3])), widget.NewLabelWithData(horse3Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[4])), widget.NewLabelWithData(horse4Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[5])), widget.NewLabelWithData(horse5Name)),
	)

	player := widget.NewEntry()
	queryByPlayer := container.NewGridWithColumns(2,
		container.NewGridWithColumns(3,
			widget.NewLabel("查询玩家马匹信息"),
			player,
			widget.NewButton("查询", func() {
				g.queryHorse(player.Text)
			})),
		container.NewGridWithColumns(3,
			container.NewGridWithColumns(1),
			container.NewGridWithColumns(1),
			widget.NewButton("清屏", func() {
				display("")
			})),
	)

	confirm := container.NewGridWithColumns(6,
		widget.NewLabel("查询自己马匹信息"),
		widget.NewButton("查询", func() {
			g.queryHorse("")
		}),
		container.NewGridWithColumns(1),
		container.NewGridWithColumns(1),
		container.NewGridWithColumns(1),
		widget.NewButton("保存", func() {
			g.update()
		}))

	box := container.NewGridWithColumns(1,
		container.NewGridWithRows(4,
			container.NewGridWithRows(2, widget.NewLabel("配置选项"), checkConf),
			container.NewGridWithRows(2, widget.NewLabel("马匹名称"), horseConf),
			container.NewGridWithRows(2, widget.NewLabel("发送哪些马匹信息"), horseCheck),
			container.NewGridWithRows(2, confirm, queryByPlayer),
		),
		container.NewScroll(g.output))

	w.SetContent(box)
	w.Resize(resize(1000, 600))
	w.Show()
}

func (g *gui) queryHorse(player string) {
	name, score, kda, horse, err := g.p.queryBySummonerName(player)
	if err != nil {
		Append(err)
		return
	}
	Append(fmt.Sprintf("本局%s：%s 得分：%.2f 近期KDA：%s", name, horse, score, kda))
}

func (g *gui) update() {
	err := g.p.UpdateClientConf(g.conf)
	if err != nil {
		Append("保存失败", err)
		return
	}
	Append("保存成功")
}

func display(newtext string) {
	GetLol().output.SetText(newtext)
}

func Append(newtext ...interface{}) {
	original := GetLol().output.Text
	text := fmt.Sprint(newtext)
	text = text[1 : len(text)-1]
	GetLol().output.SetText(original + fmt.Sprintf("%s : %s\n", time.Now().Format(layout), text))
}

func resize(w float32, h float32) fyne.Size {
	return fyne.NewSize(w, h)
}

func newLol() *gui {
	bootstrap.InitApp()
	defer global.Cleanup()
	prophet := NewProphet()
	return &gui{
		conf: getAll(),
		p:    prophet,
	}
}

func GetLol() *gui {
	once.Do(func() {
		lol = newLol()
	})
	return lol
}
