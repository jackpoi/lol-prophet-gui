package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type UpdateClientConfReq struct {
	AutoAcceptGame                 bool     `json:"autoAcceptGame"`
	AutoPickChampID                int      `json:"autoPickChampID"`
	AutoBanChampID                 int      `json:"autoBanChampID"`
	AutoSendTeamHorse              bool     `json:"autoSendTeamHorse"`
	ShouldSendSelfHorse            bool     `json:"shouldSendSelfHorse"`
	HorseNameConf                  []string `json:"horseNameConf"`
	ChooseSendHorseMsg             []bool   `json:"chooseSendHorseMsg"`
	ChooseChampSendMsgDelaySec     int      `json:"chooseChampSendMsgDelaySec"`
	ShouldInGameSaveMsgToClipBoard bool     `json:"shouldInGameSaveMsgToClipBoard"`
	ShouldAutoOpenBrowser          bool     `json:"shouldAutoOpenBrowser"`
}

type summonerNameReq struct {
	SummonerName string `json:"summonerName"`
}

const (
	context = "127.0.0.1:4396"
	version = "v1"
)

func getAll() *UpdateClientConfReq {
	//resp, err := http.Post("http://127.0.0.1:4396/v1/config/getAll")
	//if err != nil {
	//	return nil
	//}
	//bts, _ := io.ReadAll(resp.Body)
	//conf := &UpdateClientConfReq{}
	//json.Unmarshal(bts, conf)
	//return conf
	return &UpdateClientConfReq{
		AutoAcceptGame:                 true,
		AutoPickChampID:                0,
		AutoBanChampID:                 0,
		AutoSendTeamHorse:              true,
		ShouldSendSelfHorse:            true,
		HorseNameConf:                  []string{"汗血宝马", "上等马", "中等马", "下等马", "牛马", "没有马"},
		ChooseSendHorseMsg:             []bool{true, true, true, true, true, true},
		ChooseChampSendMsgDelaySec:     3,
		ShouldInGameSaveMsgToClipBoard: true,
		ShouldAutoOpenBrowser:          false,
	}
}

func (conf *UpdateClientConfReq) update() error {
	body, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	_, err = http.Post("http://127.0.0.1:4396/v1/config/update", "", bytes.NewBuffer(body))
	return err
}

func queryBySummonerName(name string) (string, error) {
	body, err := json.Marshal(summonerNameReq{name})
	if err != nil {
		return "", err
	}
	resp, err := http.Post("http://127.0.0.1:4396/v1/horse/queryBySummonerName", "", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	// todo
	//resp
	return string(result), err
}
