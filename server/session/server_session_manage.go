package session

import (
	"github.com/astaxie/beego"
	"github.com/beego/beego/v2/core/logs"
	"dewfn.com/nps/lib/cache"
	"dewfn.com/nps/lib/common"
	"math/rand"
	"path/filepath"
	"strings"
)

var MyIp string

var outsideIp string

func ServerRegister() {

	if beego.AppConfig.DefaultBool("net_outside", false) {
		if b, err := common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "conf", "outside.conf")); err != nil {
			panic(err)
		} else {
			outsideIp = strings.TrimSpace(string(b))
		}
	} else {
		outsideIp = MyIp + ":" + beego.AppConfig.String("bridge_port")
	}
	cache.GetRedis().HSet(cache.LIVE_SERVER_LIST_KEY, MyIp, outsideIp)
}
func ServerHeartbeat() {
	cache.GetRedis().Set(cache.LIVE_SERVER_KEY+MyIp, 1, cache.LIVE_SERVER_TIMEOUT)
}
func ServerIsConnected(serverIp string) bool {
	status, err := cache.GetRedis().Exists(cache.LIVE_SERVER_KEY + serverIp).Result()
	return err == nil && status > 0

}

func GetLiveServer() (liveServerMap map[string]string) {
	liveServerMap = make(map[string]string)
	allServerList, err := cache.GetRedis().HGetAll(cache.LIVE_SERVER_LIST_KEY).Result()
	if err != nil && allServerList != nil {
		logs.Error("获取存活服务器列表失败", err)
		return nil
	}
	var status int
	for k, v := range allServerList {
		status, err = cache.GetRedis().Get(cache.LIVE_SERVER_KEY + k).Int()
		if status == 1 && err == nil {
			liveServerMap[k] = v
		} else {
			//剔除没有心跳的服务器， 但是服务器重连是没有注册的
			//cache.GetRedis().SRem(cache.LIVE_SERVER_LIST_KEY,serverIp)
		}
	}
	return
}
func DistributionLiveServer(oldServerIp string) (serverIp string, outIp string) {
	serverMaps := GetLiveServer()
	if len(serverMaps) > 0 {
		if outIp = serverMaps[oldServerIp]; oldServerIp != "" && outIp != "" {
			//分配过服务端IP就直接用
			serverIp = oldServerIp
			return
		}
		//没有分过话，就随机一条
		randomIndex := rand.Intn(len(serverMaps))
		i := 0
		for k, v := range serverMaps {
			if randomIndex == i {
				serverIp = k
				outIp = v
				return
			}
			i++
		}
	}
	return
}
