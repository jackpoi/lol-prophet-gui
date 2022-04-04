package lol_prophet_gui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/beastars1/lol-prophet-gui/bootstrap"
	"github.com/beastars1/lol-prophet-gui/champion"
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

	championSelect := widget.NewSelect(champion.GetChampions(), func(s string) {
		g.conf.AutoPickChampID = champion.GetKeyByName(s)
	})
	championSelect.SetSelectedIndex(0)
	checkConf := container.NewGridWithColumns(3,
		container.NewHBox(
			widget.NewCheckWithData("自动接受对局", binding.BindBool(&g.conf.AutoAcceptGame)),
			widget.NewCheckWithData("发送自己马匹信息", binding.BindBool(&g.conf.ShouldSendSelfHorse)),
		),
		container.NewHBox(
			widget.NewCheckWithData("", binding.BindBool(&g.conf.AutoSendTeamHorse)),
			widget.NewLabel("选择英雄"),
			widget.NewEntryWithData(binding.IntToString(binding.BindInt(&g.conf.ChooseChampSendMsgDelaySec))),
			widget.NewLabel("秒后自动发送"),
		),
		container.NewHBox(
			widget.NewLabel("自动选择英雄"),
			championSelect,
		),
	)

	horse0Name := binding.BindString(&g.conf.HorseNameConf[0])
	horse1Name := binding.BindString(&g.conf.HorseNameConf[1])
	horse2Name := binding.BindString(&g.conf.HorseNameConf[2])
	horse3Name := binding.BindString(&g.conf.HorseNameConf[3])
	horse4Name := binding.BindString(&g.conf.HorseNameConf[4])
	horseConf := container.NewGridWithColumns(5,
		newBindEntry(horse0Name),
		newBindEntry(horse1Name),
		newBindEntry(horse2Name),
		newBindEntry(horse3Name),
		newBindEntry(horse4Name),
	)

	horseCheck := container.NewGridWithColumns(5,
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[0])), widget.NewLabelWithData(horse0Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[1])), widget.NewLabelWithData(horse1Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[2])), widget.NewLabelWithData(horse2Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[3])), widget.NewLabelWithData(horse3Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&g.conf.ChooseSendHorseMsg[4])), widget.NewLabelWithData(horse4Name)),
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

func newBindEntry(data binding.String) *widget.Entry {
	entry := widget.NewEntry()
	entry.Bind(data)
	return entry
}

func (g *gui) queryHorse(player string) {
	name, score, kda, horse, err := g.p.queryBySummonerName(player)
	if err != nil {
		Append(err)
		return
	}
	Append(fmt.Sprintf("%s：%s 得分：%.1f 近期KDA：%s", name, horse, score, kda))
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
		conf: global.GetClientConf(),
		p:    prophet,
	}
}

func GetLol() *gui {
	once.Do(func() {
		lol = newLol()
	})
	return lol
}
