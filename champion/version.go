package champion

import (
	"encoding/json"
	"github.com/beastars1/lol-prophet-gui/pkg/tool"
	"github.com/beastars1/lol-prophet-gui/services/logger"
)

type (
	version string
)

const (
	versionUrl = "https://ddragon.leagueoflegends.com/api/versions.json"
)

func GetVersions() []version {
	versions := getVersionList()
	if len(versions) > 5 {
		return versions[:5]
	} else {
		return versions[:]
	}
}

func getVersionList() []version {
	var versions []version
	body := tool.HttpGet(versionUrl)
	err := json.Unmarshal(body, &versions)
	if err != nil {
		logger.Error("json cant unmarshal", err)
		return versions
	}
	return versions
}
