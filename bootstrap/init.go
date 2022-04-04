package bootstrap

import (
	"encoding/json"
	"github.com/beastars1/lol-prophet-gui/conf"
	"github.com/beastars1/lol-prophet-gui/global"
	"github.com/beastars1/lol-prophet-gui/pkg/logger"
	"github.com/beastars1/lol-prophet-gui/pkg/tool"
	"github.com/beastars1/lol-prophet-gui/pkg/windows/admin"
	"github.com/beastars1/lol-prophet-gui/services/db/enity"
	"github.com/beastars1/lol-prophet-gui/services/ws"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/now"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const (
	defaultTZ = "Asia/Shanghai"
)

func initConf() {
	_ = godotenv.Load(".env")
	if tool.IsFile(".env.local") {
		_ = godotenv.Overload(".env.local")
	}

	*global.Conf = global.DefaultAppConf
	err := configor.Load(global.Conf)
	if err != nil {
		panic(err)
	}
	err = initClientConf()
	if err != nil {
		panic(err)
	}
}

func initClientConf() (err error) {
	dbPath := conf.SqliteDBPath
	var db *gorm.DB
	var dbLogger = gormLogger.Discard
	if global.IsDevMode() {
		dbLogger = gormLogger.Default
	}
	gormCfg := &gorm.Config{
		Logger: dbLogger,
	}
	if !tool.IsFile(dbPath) {
		db, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
		bts, _ := json.Marshal(global.DefaultClientConf)
		err = db.Exec(enity.InitLocalClientSql, enity.LocalClientConfKey, string(bts)).Error
		if err != nil {
			return
		}
		*global.ClientConf = global.DefaultClientConf
	} else {
		db, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
		confItem := &enity.Config{}
		err = db.Table("config").Where("k = ?", enity.LocalClientConfKey).First(confItem).Error
		if err != nil {
			return
		}
		localClientConf := &conf.Client{}
		err = json.Unmarshal([]byte(confItem.Val), localClientConf)
		if err != nil || conf.ValidClientConf(localClientConf) != nil {
			return errors.New("本地配置错误")
		}
		global.ClientConf = localClientConf
	}
	global.SqliteDB = db
	return nil
}

func initLog(cfg *conf.LogConf) {
	writeSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filepath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	})
	if global.IsDevMode() {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeDuration = zapcore.StringDurationEncoder
	level, err := logger.Str2ZapLevel(cfg.Level)
	if err != nil {
		panic("zap level is Incorrect")
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		writeSyncer,
		zap.NewAtomicLevelAt(level),
	)
	global.Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
}

func InitApp() {
	admin.MustRunWithAdmin()
	initConsole()
	initConf()
	initLog(&global.Conf.Log)
	initLib()
	initGlobal()
}

func initConsole() {
	stdIn := windows.Handle(os.Stdin.Fd())
	var consoleMode uint32
	_ = windows.GetConsoleMode(stdIn, &consoleMode)
	consoleMode = consoleMode&^windows.ENABLE_QUICK_EDIT_MODE | windows.ENABLE_EXTENDED_FLAGS
	_ = windows.SetConsoleMode(stdIn, consoleMode)
}

func initGlobal() {
	//go ...
}

func initLib() {
	_ = os.Setenv("TZ", defaultTZ)
	now.WeekStartDay = time.Monday
	go func() {
		initUserInfo()
		if global.Conf.Sentry.Enabled {
			_ = initSentry(global.Conf.Sentry.Dsn)
		}
	}()
	ws.Init()
}

func initUserInfo() {
	global.SetUserInfo(global.UserInfo{
		IP: getIp(),
		// Mac:   windows.GetMac(),
		// CpuID: windows.GetCpuID(),
	})
}

func getIp() string {
	ip := "unknown"
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.ip.sb/ip", nil)
	req.Header.Set("user-agent", "lol")

	resp, err := client.Do(req)
	if err != nil {
		return ip
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		bts, _ := io.ReadAll(resp.Body)
		ip = strings.Trim(string(bts), "\n")
	}
	return ip
}

func initSentry(dsn string) error {
	isDebugMode := global.IsDevMode()
	sampleRate := 1.0
	if !isDebugMode {
		sampleRate = 1.0
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:        dsn,
		Debug:      isDebugMode,
		SampleRate: sampleRate,
		//Release:     lol_prophet_gui.Commit,
		Environment: global.GetEnv(),
	})
	if err == nil {
		global.Cleanups["sentryFlush"] = func() error {
			sentry.Flush(2 * time.Second)
			return nil
		}
		userInfo := global.GetUserInfo()
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("lol", map[string]interface{}{
				"ip":      userInfo.IP,
				"version": global.AppBuildInfo.Version,
				// "mac":   userInfo.Mac,
				// "cpuID": userInfo.CpuID,
			})
			scope.SetUser(sentry.User{
				// ID:        userInfo.Mac,
				IPAddress: userInfo.IP,
			})
			// scope.SetExtra("cpuID", userInfo.CpuID)
		})
	}
	return err
}
