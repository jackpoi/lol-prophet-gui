package champion

import (
	"encoding/json"
	"fmt"
	"github.com/beastars1/lol-prophet-gui/pkg/tool"
	"github.com/beastars1/lol-prophet-gui/services/logger"
	"regexp"
	"strconv"
	"strings"
)

var (
	championsMap = map[string]int{}
	champions    = []string{"无"}
)

func init() {
	championList := GetChampionList("12.6.1")
	for _, v := range championList {
		championsMap[v.Name], _ = strconv.Atoi(v.Key)
		champions = append(champions, v.Name)
	}
}

func GetChampions() []string {
	return champions
}

func GetKeyByName(name string) int {
	key, ok := championsMap[name]
	if ok {
		return key
	}
	return 0
}

const (
	championListUrl = "http://ddragon.leagueoflegends.com/cdn/%s/data/zh_CN/champion.json"
)

type championInfo struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

func GetChampionList(version string) []championInfo {
	body := tool.HttpGet(fmt.Sprintf(championListUrl, version))
	expr := `"key([\s\S]*?)blurb"`
	str := regexpMatch(string(body), expr)
	return formatChampion(str)
}

func formatChampion(original [][]string) []championInfo {
	builder := strings.Builder{}
	builder.WriteString("[")
	for i, s := range original {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("{")
		builder.WriteString(strings.TrimSuffix(s[0], `,"blurb"`))
		builder.WriteString("}")
	}
	builder.WriteString("]")
	var champions []championInfo
	s := builder.String()
	err := json.Unmarshal([]byte(s), &champions)
	if err != nil {
		logger.Error("json cant unmarshal", err)
		return champions
	}
	return champions
}

func regexpMatch(text, expr string) [][]string {
	reg, err := regexp.Compile(expr)
	if err != nil {
		logger.Error("regexp cant compile", err)
		return nil
	}
	return reg.FindAllStringSubmatch(text, -1)
}
