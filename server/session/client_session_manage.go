package session

import (
	"encoding/json"
	"github.com/beego/beego/v2/core/logs"
	"dewfn.com/nps/lib/cache"
	"strconv"
	"time"
)

type ClientSession struct {
	ServerIp    string
	ConnectTime time.Time
	Version     string
}

func ClientIsConnected(clientId int64) bool {
	clientSession, err := GetServerByClient(clientId)
	if err == nil && clientSession != nil && clientSession.ServerIp != "" {
		return true
	}
	return false
}

func ClientConnecting(clientId int64, version string) bool {
	bd, _ := json.Marshal(ClientSession{ServerIp: MyIp, ConnectTime: time.Now(), Version: version})
	cache.GetRedis().Set(cache.CLIENT_SESSION_KEY+strconv.FormatInt(clientId, 10), string(bd), cache.CLIENT_SESSION_TIMEOUT)
	return true
}

func ClientHeartbeat(clientId int64) bool {
	cache.GetRedis().Expire(cache.CLIENT_SESSION_KEY+strconv.FormatInt(clientId, 10), cache.CLIENT_SESSION_TIMEOUT)
	return true
}

func GetServerByClient(clientId int64) (clientSession *ClientSession, err error) {
	data, err := cache.GetRedis().Get(cache.CLIENT_SESSION_KEY + strconv.FormatInt(clientId, 10)).Bytes()
	if err == nil && data != nil {
		err = json.Unmarshal((data), &clientSession)
		if err != nil {
			logs.Error(err)
		}
	}
	return
}
func IsLocalServer(serverIp string) bool {
	if serverIp == MyIp {
		return true
	}
	return false
}

func RemoveClient(clientId int64) {
	cache.GetRedis().Del(cache.CLIENT_SESSION_KEY + strconv.FormatInt(clientId, 10))
}
