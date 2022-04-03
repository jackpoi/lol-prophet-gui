package lol_prophet_gui

import (
	"bytes"
	"encoding/json"
	"github.com/beastars1/lol-prophet-gui/conf"
	"github.com/beastars1/lol-prophet-gui/global"
	ginApp "github.com/beastars1/lol-prophet-gui/pkg/gin"
	"github.com/beastars1/lol-prophet-gui/services/db/models"
	"github.com/beastars1/lol-prophet-gui/services/lcu"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type (
	Api struct {
		p *Prophet
	}
	summonerNameReq struct {
		SummonerName string `json:"summonerName"`
	}
)

func (api Api) ProphetActiveMid(c *gin.Context) {
	app := ginApp.GetApp(c)
	if !api.p.lcuActive {
		app.ErrorMsg("请检查lol客户端是否已启动")
		return
	}
	c.Next()
}
func (api Api) QueryHorseBySummonerName(c *gin.Context) {
	app := ginApp.GetApp(c)
	d := &summonerNameReq{}
	if err := c.ShouldBind(d); err != nil {
		app.ValidError(err)
		return
	}
	summonerName := strings.TrimSpace(d.SummonerName)
	var summonerID int64 = 0
	if summonerName == "" {
		if api.p.currSummoner == nil {
			app.ErrorMsg("系统错误")
			return
		}
		// 如果为空，查询自己的分数
		summonerID = api.p.currSummoner.SummonerId
	} else {
		info, err := lcu.QuerySummonerByName(summonerName)
		if err != nil || info.SummonerId <= 0 {
			app.ErrorMsg("未查询到召唤师")
			return
		}
		summonerID = info.SummonerId
	}
	scoreInfo, err := GetUserScore(summonerID)
	if err != nil {
		app.CommonError(err)
		return
	}
	scoreCfg := global.GetScoreConf()
	clientCfg := global.GetClientConf()
	var horse string
	for i, v := range scoreCfg.Horse {
		if scoreInfo.Score >= v.Score {
			horse = clientCfg.HorseNameConf[i]
			break
		}
	}
	app.Data(gin.H{
		"score":   scoreInfo.Score,
		"currKDA": scoreInfo.CurrKDA,
		"horse":   horse,
	})
}

func (api Api) CopyHorseMsgToClipBoard(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Success()
}
func (api Api) GetAllConf(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(global.GetClientConf())
}
func (api Api) UpdateClientConf(c *gin.Context) {
	app := ginApp.GetApp(c)
	d := &conf.UpdateClientConfReq{}
	if err := c.ShouldBind(d); err != nil {
		app.ValidError(err)
		return
	}
	cfg := global.SetClientConf(*d)
	bts, _ := json.Marshal(cfg)
	m := models.Config{}
	err := m.Update(models.LocalClientConfKey, string(bts))
	if err != nil {
		app.CommonError(err)
		return
	}
	app.Success()
}
func (api Api) DevHand(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(gin.H{
		"buffge": 23456,
	})
}
func (api Api) GetAppInfo(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(global.AppBuildInfo)
}
func (api Api) GetLcuAuthInfo(c *gin.Context) {
	app := ginApp.GetApp(c)
	port, token, err := lcu.GetLolClientApiInfo()
	if err != nil {
		app.CommonError(err)
		return
	}
	app.Data(gin.H{
		"token": token,
		"port":  port,
	})
}

const (
	localUrl = "127.0.0.1:4396"
	version  = "v1"
)

func getAll() *conf.Client {
	//resp, err := http.Post("http://127.0.0.1:4396/v1/config/getAll")
	//if err != nil {
	//	return nil
	//}
	//bts, _ := io.ReadAll(resp.Body)
	//conf := &UpdateClientConfReq{}
	//json.Unmarshal(bts, conf)
	//return conf
	return &global.DefaultClientConf
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
