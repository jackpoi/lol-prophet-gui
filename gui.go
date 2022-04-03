package lol_prophet_gui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/beastars1/lol-prophet-gui/conf"
	"time"
)

const (
	layout = "2006-01-02 15:01:05"
)

type lol struct {
	output *widget.Entry
	conf   *conf.Client
	window fyne.Window
}

func (l *lol) LoadUI(app fyne.App) {
	l.output = widget.NewMultiLineEntry()
	l.output.Resize(resize(800, 300))
	l.output.TextStyle.Monospace = false

	w := app.NewWindow("LOL 先知")

	entry := widget.NewEntryWithData(binding.IntToString(binding.BindInt(&l.conf.ChooseChampSendMsgDelaySec)))
	checkConf := container.NewGridWithColumns(4,
		widget.NewCheckWithData("自动接受对局", binding.BindBool(&l.conf.AutoAcceptGame)),
		widget.NewCheckWithData("选择英雄界面自动发送", binding.BindBool(&l.conf.AutoSendTeamHorse)),
		widget.NewCheckWithData("发送自己马匹信息", binding.BindBool(&l.conf.ShouldSendSelfHorse)),
		container.NewHBox(widget.NewLabel("选择英雄"), entry, widget.NewLabel("秒后发送")),
	)

	horse0Name := binding.BindString(&l.conf.HorseNameConf[0])
	horse1Name := binding.BindString(&l.conf.HorseNameConf[1])
	horse2Name := binding.BindString(&l.conf.HorseNameConf[2])
	horse3Name := binding.BindString(&l.conf.HorseNameConf[3])
	horse4Name := binding.BindString(&l.conf.HorseNameConf[4])
	horse5Name := binding.BindString(&l.conf.HorseNameConf[5])
	horseConf := container.NewGridWithColumns(6,
		widget.NewEntryWithData(horse0Name),
		widget.NewEntryWithData(horse1Name),
		widget.NewEntryWithData(horse2Name),
		widget.NewEntryWithData(horse3Name),
		widget.NewEntryWithData(horse4Name),
		widget.NewEntryWithData(horse5Name),
	)

	horseCheck := container.NewGridWithColumns(6,
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&l.conf.ChooseSendHorseMsg[0])), widget.NewLabelWithData(horse0Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&l.conf.ChooseSendHorseMsg[1])), widget.NewLabelWithData(horse1Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&l.conf.ChooseSendHorseMsg[2])), widget.NewLabelWithData(horse2Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&l.conf.ChooseSendHorseMsg[3])), widget.NewLabelWithData(horse3Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&l.conf.ChooseSendHorseMsg[4])), widget.NewLabelWithData(horse4Name)),
		container.NewHBox(widget.NewCheckWithData("", binding.BindBool(&l.conf.ChooseSendHorseMsg[5])), widget.NewLabelWithData(horse5Name)),
	)

	player := widget.NewEntry()
	queryByPlayer := container.NewGridWithColumns(2,
		container.NewGridWithColumns(3,
			widget.NewLabel("查询玩家马匹信息"),
			player,
			widget.NewButton("查询", func() {
				horse, err := queryBySummonerName(player.Text)
				l.append(player.Text)
				if err != nil {
					l.append(err)
					return
				}
				l.append(horse)
			})),
		container.NewGridWithColumns(3,
			container.NewGridWithColumns(1),
			container.NewGridWithColumns(1),
			widget.NewButton("清屏", func() {
				l.display("")
			})),
	)

	confirm := container.NewGridWithColumns(6,
		widget.NewLabel("查询自己马匹信息"),
		widget.NewButton("查询", func() {
			horse, err := queryBySummonerName("")
			if err != nil {
				l.append(err)
				return
			}
			l.append(horse)
		}),
		container.NewGridWithColumns(1),
		container.NewGridWithColumns(1),
		container.NewGridWithColumns(1),
		widget.NewButton("保存", func() {
			l.append(l.conf.ChooseChampSendMsgDelaySec)
			err := l.conf.Update()
			if err != nil {
				l.append("更新配置失败\n")
				return
			}
		}))

	box := container.NewGridWithColumns(1,
		container.NewGridWithRows(4,
			container.NewGridWithRows(2, widget.NewLabel("配置选项"), checkConf),
			container.NewGridWithRows(2, widget.NewLabel("马匹名称"), horseConf),
			container.NewGridWithRows(2, widget.NewLabel("发送哪些马匹信息"), horseCheck),
			container.NewGridWithRows(2, confirm, queryByPlayer),
		),
		container.NewScroll(l.output))

	w.SetContent(box)
	w.Resize(resize(1000, 600))
	w.Show()
}

func (l *lol) display(newtext string) {
	l.output.SetText(newtext)
}

func (l *lol) append(newtext ...interface{}) {
	original := l.output.Text
	text := fmt.Sprint(newtext)
	text = text[1 : len(text)-1]
	l.output.SetText(original + fmt.Sprintf("%s : %s", time.Now().Format(layout), text))
}

func resize(w float32, h float32) fyne.Size {
	return fyne.NewSize(w, h)
}

func NewLol() *lol {
	return &lol{
		conf: getAll(),
	}
}
