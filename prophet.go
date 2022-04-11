package lol_prophet_gui

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/beastars1/lol-prophet-gui/champion"
	"github.com/beastars1/lol-prophet-gui/conf"
	"github.com/beastars1/lol-prophet-gui/global"
	"github.com/beastars1/lol-prophet-gui/services/db/enity"
	"github.com/beastars1/lol-prophet-gui/services/lcu"
	"github.com/beastars1/lol-prophet-gui/services/lcu/models"
	"github.com/beastars1/lol-prophet-gui/services/logger"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	lcuWsEvt  string
	GameState string
	Prophet   struct {
		ctx          context.Context
		opts         *options
		httpSrv      *http.Server
		lcuPort      int
		lcuToken     string
		lcuActive    bool
		currSummoner *lcu.CurrSummoner
		cancel       func()
		mu           *sync.Mutex
		GameState    GameState
	}
	wsMsg struct {
		Data      interface{} `json:"data"`
		EventType string      `json:"event_type"`
		Uri       string      `json:"uri"`
	}
	options struct {
		debug       bool
		enablePprof bool
	}
)

const (
	onJsonApiEventPrefixLen              = len(`[8,"OnJsonApiEvent",`)
	gameFlowChangedEvt          lcuWsEvt = "/lol-gameflow/v1/gameflow-phase"
	champSelectUpdateSessionEvt lcuWsEvt = "/lol-champ-select/v1/session"
)

// gameState
const (
	GameStateNone        GameState = "none"
	GameStateChampSelect GameState = "champSelect"
	GameStateReadyCheck  GameState = "ReadyCheck"
	GameStateInGame      GameState = "inGame"
	GameStateOther       GameState = "other"
)

var (
	defaultOpts = &options{
		debug:       false,
		enablePprof: true,
	}
)

func NewProphet(opts ...ApplyOption) *Prophet {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Prophet{
		ctx:       ctx,
		cancel:    cancel,
		mu:        &sync.Mutex{},
		opts:      defaultOpts,
		GameState: GameStateNone,
	}
	if global.IsDevMode() {
		opts = append(opts, WithDebug())
	} else {
		opts = append(opts, WithProd())
	}
	for _, fn := range opts {
		fn(p.opts)
	}
	return p
}

func (p *Prophet) Run() {
	go p.MonitorStart()
	go p.captureStartMessage()
	Append(fmt.Sprintf("%s已启动，当前英雄列表版本:%s", global.AppName, champion.Version))
}

func (p *Prophet) isLcuActive() bool {
	return p.lcuActive
}

func (p *Prophet) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	// stop all task
	return nil
}

func (p *Prophet) MonitorStart() {
	for {
		if !p.isLcuActive() {
			port, token, err := lcu.GetLolClientApiInfo()
			if err != nil {
				if !errors.Is(lcu.ErrLolProcessNotFound, err) {
					logger.Error("获取lcu info 失败", zap.Error(err))
				}
				time.Sleep(time.Second)
				continue
			}
			p.initLcuClient(port, token)
			err = p.initGameFlowMonitor(port, token)
			if err != nil {
				logger.Debug("游戏流程监视器 err:", zap.Error(err))
			}
			p.lcuActive = false
			p.currSummoner = nil
		}
		time.Sleep(time.Second)
	}
}

func (p *Prophet) initLcuClient(port int, token string) {
	lcu.InitCli(port, token)
}

func (p *Prophet) initGameFlowMonitor(port int, authPwd string) error {
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	rawUrl := fmt.Sprintf("wss://127.0.0.1:%d/", port)
	header := http.Header{}
	authSecret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("riot:%s", authPwd)))
	header.Set("Authorization", "Basic "+authSecret)
	u, _ := url.Parse(rawUrl)
	logger.Debug(fmt.Sprintf("connect to lcu %s", u.String()))
	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return err
	}
	defer c.Close()
	err = retry.Do(func() error {
		currSummoner, err := lcu.GetCurrSummoner()
		if err == nil {
			p.currSummoner = currSummoner
		}
		return err
	}, retry.Attempts(5), retry.Delay(time.Second))
	if err != nil {
		return errors.New("获取当前召唤师信息失败:" + err.Error())
	}
	p.lcuActive = true

	_ = c.WriteMessage(websocket.TextMessage, []byte("[5, \"OnJsonApiEvent\"]"))
	for {
		msgType, message, err := c.ReadMessage()
		if err != nil {
			logger.Debug("lol事件监控读取消息失败", zap.Error(err))
			return err
		}
		msg := &wsMsg{}
		if msgType != websocket.TextMessage || len(message) < onJsonApiEventPrefixLen+1 {
			continue
		}
		_ = json.Unmarshal(message[onJsonApiEventPrefixLen:len(message)-1], msg)
		//Append("lol事件监控读取消息 -> ", msg.Uri)
		switch msg.Uri {
		case string(gameFlowChangedEvt):
			gameFlow, ok := msg.Data.(string)
			if !ok {
				continue
			}
			p.onGameFlowUpdate(gameFlow)
		case string(champSelectUpdateSessionEvt):
			bts, err := json.Marshal(msg.Data)
			if err != nil {
				continue
			}
			sessionInfo := &lcu.ChampSelectSessionInfo{}
			err = json.Unmarshal(bts, sessionInfo)
			if err != nil {
				logger.Warn("解析结构体失败", err)
				continue
			}
			go func() {
				_ = p.onChampSelectSessionUpdate(sessionInfo)
			}()
		default:

		}
	}
}

func (p *Prophet) onGameFlowUpdate(gameFlow string) {
	//Append("切换状态:" + gameFlow)
	switch gameFlow {
	case string(models.GameFlowChampionSelect):
		Append("进入英雄选择阶段，正在计算分数")
		sentry.CaptureMessage("进入英雄选择阶段，正在计算分数")
		p.updateGameState(GameStateChampSelect)
		go p.ChampionSelectStart()
	case string(models.GameFlowNone):
		p.updateGameState(GameStateNone)
	case string(models.GameFlowInProgress):
		p.updateGameState(GameStateInGame)
		go p.CalcEnemyTeamScore()
	case string(models.GameFlowReadyCheck):
		p.updateGameState(GameStateReadyCheck)
		clientCfg := global.GetClientConf()
		if clientCfg.AutoAcceptGame {
			go p.AcceptGame()
		}
	default:
		p.updateGameState(GameStateOther)
	}
}

func (p *Prophet) updateGameState(state GameState) {
	p.mu.Lock()
	p.GameState = state
	p.mu.Unlock()
}

func (p *Prophet) getGameState() GameState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.GameState
}

func (p *Prophet) captureStartMessage() {
	for i := 0; i < 5; i++ {
		if global.GetUserInfo().IP != "" {
			break
		}
		time.Sleep(time.Second * 2)
	}
	sentry.CaptureMessage(global.AppName + "已启动")
}

// ChampionSelectStart 选择英雄时进行核心逻辑处理：获取人员、计算得分、发送信息
func (p Prophet) ChampionSelectStart() {
	clientCfg := global.GetClientConf()
	sendConversationMsgDelayCtx, cancel := context.WithTimeout(context.Background(),
		time.Second*time.Duration(clientCfg.ChooseChampSendMsgDelaySec))
	defer cancel()
	var conversationID string
	var summonerIDList []int64
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		// 获取队伍所有玩家信息
		conversationID, summonerIDList, _ = getTeamUsers()
		if len(summonerIDList) != 5 {
			continue
		}
	}

	logger.Debug("队伍人员列表:", zap.Any("summonerIDList", summonerIDList))
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerIDMapScore := map[int64]lcu.UserScore{}
	mu := sync.Mutex{}
	for _, summonerID := range summonerIDList {
		summonerID := summonerID
		g.Go(func() error {
			actScore, err := GetUserScore(summonerID)
			if err != nil {
				logger.Error("计算玩家分数失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	scoreCfg := global.GetScoreConf()
	allMsg := ""
	mergedMsg := ""
	// 发送到选人界面
	for _, scoreInfo := range summonerIDMapScore {
		var horse string
		horseIdx := 0
		for i, v := range scoreCfg.Horse {
			if scoreInfo.Score >= v.Score {
				horse = clientCfg.HorseNameConf[i]
				horseIdx = i
				break
			}
		}
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(scoreInfo.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", scoreInfo.CurrKDA[i][0], scoreInfo.CurrKDA[i][1],
				scoreInfo.CurrKDA[i][2]))
		}
		currKDAMsg := currKDASb.String()
		if len(currKDAMsg) > 0 {
			currKDAMsg = currKDAMsg[:len(currKDAMsg)-1]
		}

		msg := fmt.Sprintf("本局%s：%s 得分：%.1f  近期KDA：%s", horse, scoreInfo.SummonerName, scoreInfo.Score, currKDAMsg)
		//log.Printf(msg)
		Append(msg)
		<-sendConversationMsgDelayCtx.Done()
		if clientCfg.AutoSendTeamHorse {
			mergedMsg += msg + "\n"
		}
		if !clientCfg.AutoSendTeamHorse {
			if !scoreCfg.MergeMsg && !clientCfg.ShouldSendSelfHorse && p.currSummoner != nil &&
				scoreInfo.SummonerID == p.currSummoner.SummonerId {
				continue
			}
			allMsg += msg + "\n"
			mergedMsg += msg + "\n"
			continue
		}
		if !clientCfg.ShouldSendSelfHorse && p.currSummoner != nil &&
			scoreInfo.SummonerID == p.currSummoner.SummonerId {
			continue
		}
		if !clientCfg.ChooseSendHorseMsg[horseIdx] {
			continue
		}
		if scoreCfg.MergeMsg {
			continue
		}
		_ = SendConversationMsg(msg, conversationID)
		time.Sleep(time.Millisecond * 1500)
	}
	if !clientCfg.AutoSendTeamHorse {
		Append("已将队伍马匹信息复制到剪切板")
		_ = clipboard.WriteAll(allMsg)
		return
	}
	if scoreCfg.MergeMsg {
		_ = SendConversationMsg(mergedMsg, conversationID)
	}
}

func (p Prophet) AcceptGame() {
	_ = lcu.AcceptGame()
}

func (p Prophet) CalcEnemyTeamScore() {
	// 获取当前游戏进程
	session, err := lcu.QueryGameFlowSession()
	if err != nil {
		return
	}
	if session.Phase != models.GameFlowInProgress {
		return
	}
	if p.currSummoner == nil {
		return
	}
	selfID := p.currSummoner.SummonerId
	selfTeamUsers, enemyTeamUsers := getAllUsersFromSession(selfID, session)
	_ = selfTeamUsers
	summonerIDList := enemyTeamUsers

	logger.Debug("敌方队伍人员列表:", zap.Any("summonerIDList", summonerIDList))
	if len(summonerIDList) == 0 {
		return
	}
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerIDMapScore := map[int64]lcu.UserScore{}
	mu := sync.Mutex{}
	for _, summonerID := range summonerIDList {
		summonerID := summonerID
		g.Go(func() error {
			actScore, err := GetUserScore(summonerID)
			if err != nil {
				logger.Error("计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	// 根据所有用户的分数判断小代上等马中等马下等马
	for _, score := range summonerIDMapScore {
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(score.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", score.CurrKDA[i][0], score.CurrKDA[i][1],
				score.CurrKDA[i][2]))
		}
	}
	clientCfg := global.GetClientConf()
	scoreCfg := global.GetScoreConf()
	allMsg := ""
	// 发送到选人界面
	for _, scoreInfo := range summonerIDMapScore {
		time.Sleep(time.Second / 2)
		var horse string
		// horseIdx := 0
		for i, v := range scoreCfg.Horse {
			if scoreInfo.Score >= v.Score {
				horse = clientCfg.HorseNameConf[i]
				// horseIdx = i
				break
			}
		}
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(scoreInfo.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", scoreInfo.CurrKDA[i][0], scoreInfo.CurrKDA[i][1],
				scoreInfo.CurrKDA[i][2]))
		}
		currKDAMsg := currKDASb.String()
		if len(currKDAMsg) > 0 {
			currKDAMsg = currKDAMsg[:len(currKDAMsg)-1]
		}
		msg := fmt.Sprintf("敌方%s：%s 得分：%.1f  近期KDA：%s", horse, scoreInfo.SummonerName, scoreInfo.Score, currKDAMsg)
		Append(msg)
		allMsg += msg + "\n"
	}
	_ = clipboard.WriteAll(allMsg)
}

func (p Prophet) onChampSelectSessionUpdate(sessionInfo *lcu.ChampSelectSessionInfo) error {
	isSelfPick := false
	isSelfBan := false
	userActionID := 0
	if len(sessionInfo.Actions) == 0 {
		return nil
	}
loop:
	for _, actions := range sessionInfo.Actions {
		for _, action := range actions {
			if action.ActorCellId == sessionInfo.LocalPlayerCellId && action.IsInProgress {
				userActionID = action.Id
				if action.Type == lcu.ChampSelectPatchTypePick {
					isSelfPick = true
					break loop
				} else if action.Type == lcu.ChampSelectPatchTypeBan {
					isSelfBan = true
					break loop
				}
			}
		}
	}
	clientCfg := global.GetClientConf()
	//Append("AutoPickChampID:", clientCfg.AutoPickChampID)
	//Append("userActionID:", userActionID)
	if clientCfg.AutoPickChampID > 0 && isSelfPick {
		_ = lcu.PickChampion(clientCfg.AutoPickChampID, userActionID)
	}
	if clientCfg.AutoBanChampID > 0 && isSelfBan {
		_ = lcu.BanChampion(clientCfg.AutoBanChampID, userActionID)
	}
	return nil
}

func (p Prophet) UpdateClientConf(conf *conf.Client) error {
	cfg := global.SetClientConf(conf)
	bts, _ := json.Marshal(cfg)
	m := enity.Config{}
	return m.Update(enity.LocalClientConfKey, string(bts))
}

func (p Prophet) queryBySummonerName(player string) (string, float64, string, string, error) {
	summonerName := strings.TrimSpace(player)
	var summonerID int64 = 0
	var (
		name  = ""
		score = 0.0
		kda   = ""
		horse = ""
	)

	if summonerName == "" {
		if p.currSummoner == nil {
			err := errors.New("系统错误")
			return name, score, kda, horse, err
		}
		// 如果为空，查询自己的分数
		summonerID = p.currSummoner.SummonerId
		name = p.currSummoner.DisplayName
	} else {
		info, err := lcu.QuerySummonerByName(summonerName)
		if err != nil || info.SummonerId <= 0 {
			err = errors.New("未查询到召唤师")
			return name, score, kda, horse, err
		}
		summonerID = info.SummonerId
		name = summonerName
	}
	scoreInfo, err := GetUserScore(summonerID)
	if err != nil {
		err = errors.New("系统错误")
		return name, score, kda, horse, err
	}
	score = scoreInfo.Score
	kda = kdaString(scoreInfo.CurrKDA, 5)

	scoreCfg := global.GetScoreConf()
	clientCfg := global.GetClientConf()
	for i, v := range scoreCfg.Horse {
		if scoreInfo.Score >= v.Score {
			horse = clientCfg.HorseNameConf[i]
			break
		}
	}
	return name, score, kda, horse, err
}

func kdaString(currKDA [][3]int, n int) string {
	currKDASb := strings.Builder{}
	for i := 0; i < n && i < len(currKDA); i++ {
		currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", currKDA[i][0], currKDA[i][1], currKDA[i][2]))
	}
	return currKDASb.String()
}
