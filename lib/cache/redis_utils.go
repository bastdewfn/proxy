package cache

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/beego/beego/v2/core/logs"
	"github.com/go-redis/redis"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const (
	LIVE_SERVER_LIST_KEY      = "LIVE_SERVER_LIST"
	LIVE_SERVER_KEY           = "LIVE_SERVER_"
	CLIENT_SESSION_KEY        = "CLIENT_SESSION_"
	HOST_URLMATCH_KEY         = "HOST_URLMATCH_"
	CLIENT_INFO_KEY           = "CLIENT_INFO_"
	HOST_URLMATCH_TIMEOUT     = time.Minute * 5
	LIVE_SERVER_TIMEOUT       = time.Second * 10
	CLIENT_SESSION_TIMEOUT    = time.Second * 10
	CLIENT_INFO_TIMEOUT       = time.Minute * 5
	CLIENT_INFO_ERROR_TIMEOUT = time.Minute * 5
)

// 定义一个全局变量
var redisdb *redis.ClusterClient
var once sync.Once

func GetRedis() *redis.ClusterClient {
	once.Do(func() {
		initRedis()
	})
	return redisdb
}

func initRedis() (err error) {
	appUk := beego.AppConfig.String("appUk")
	env := beego.AppConfig.String("env")
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://tccomponent.17usoft.com/tcconfigcenter6/v6/getspecifickeyvalue/%s/%s/TCBase.Cache.v2", env, appUk), nil)
	req.SetBasicAuth(appUk, appUk)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logs.Error("发起请求获取redis配置失败", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error("发起请求获取redis配置失败1", err)
		return
	}
	logs.Info("获取redis配置为 " + string(body))
	confgList := make([]*map[string]*[]*interface{}, 0)
	addrs := make([]string, 0)
	json.Unmarshal(body, &confgList)

	list := (*(*confgList[0])["instances"])

	for _, instances := range list {
		v := (*instances).(map[string]interface{})

		addrs = append(addrs, v["ip"].(string))
	}
	redisdb = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: addrs, // 指定
	})
	_, err = redisdb.Ping().Result()
	if err == nil {
		logs.Info("redis连接成功")
	} else {
		logs.Error("redis连接失败", err)
	}
	return
}
