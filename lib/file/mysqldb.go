package file

import (
	"encoding/json"
	"errors"
	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
	"dewfn.com/nps/lib/cache"
	"dewfn.com/nps/lib/common"
	"dewfn.com/nps/lib/crypt"
	"dewfn.com/nps/lib/global"
	"dewfn.com/nps/lib/rate"
	//"dewfn.com/nps/lib/services"
	"dewfn.com/nps/server/session"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type MysqlDbUtils struct {
	mysqlDb *gorm.DB
}

var (
	mysqlDb   *MysqlDbUtils
	onceMysql sync.Once
)

//init csv from file
func GetMysqlDb() *MysqlDbUtils {
	onceMysql.Do(func() {
		mysqlDb = &MysqlDbUtils{mysqlDb: global.App.GetDb()["*"]}

	})
	return mysqlDb
}

func (s *MysqlDbUtils) GetClientList() []*Client {
	list := make([]*Client, 0)

	client := &Client{}

	err := s.mysqlDb.Model(client).Where("status>-1").Find(&list).Error
	if err != nil {
		panic(err)
	}
	for _, client := range list {
		client.ToJSONObject()
	}
	return list
}

func (s *MysqlDbUtils) GetClientPage(start, length int, search, sort, order string, clientId int64) ([]*Client, int64) {
	list := make([]*Client, 0)

	var count int64

	tx := s.mysqlDb.Model(&Client{}).Where("status>-1")
	if clientId != 0 {
		tx.Where("id=?", clientId)
	}
	if search != "" {
		tx.Where("(id=? or verify_key like ? or remark like ?)", common.GetIntNoErrByStr(search), "%"+search+"%", "%"+search+"%")
	}
	err := tx.Count(&count).Error
	if err != nil {
		panic(err)
	}
	if sort == "" {
		sort = "id "
	}
	err = tx.Order(sort + order).Offset(start).Limit(length).Find(&list).Error
	if err != nil {
		panic(err)
	}

	for _, client := range list {
		client.ToJSONObject()
		clientSession, _ := session.GetServerByClient(client.Id)
		if clientSession != nil {
			client.ServerIp = clientSession.ServerIp
			client.Version = clientSession.Version
			client.IsConnect = true
			client.ConnectTime = clientSession.ConnectTime
			//service := services.ClientService{}
			//service.FullClientRealRateFlow(client, false)
		}
	}
	return list, count
}

func (s *MysqlDbUtils) GetIdByVerifyKey(vKey string, addr string, ver string, serverIp string) (id int64, err error) {

	list := make([]*Client, 0)
	err = s.mysqlDb.Where("status=? and verify_key=?", 1, vKey).Find(&list).Error
	if err != nil {
		return
	}

	var exist bool
	for _, client := range list {
		if common.Getverifyval(client.VerifyKey) == vKey {
			client.Addr = common.GetIpByAddr(addr)
			id = client.Id
			exist = true
			s.UpdateClientAddress(id, client.Addr, ver, serverIp)
			break
		}
	}
	if exist {
		return
	}
	return 0, errors.New("not found")
}

//func (s *MysqlDbUtils) NewTask(t *Tunnel) (err error) {
//	s.JsonDb.Tasks.Range(func(key, value interface{}) bool {
//		v := value.(*Tunnel)
//		if (v.Mode == "secret" || v.Mode == "p2p") && v.Password == t.Password {
//			err = errors.New(fmt.Sprintf("secret mode keys %s must be unique", t.Password))
//			return false
//		}
//		return true
//	})
//	if err != nil {
//		return
//	}
//	t.Flow = new(Flow)
//	s.JsonDb.Tasks.Store(t.Id, t)
//	s.JsonDb.StoreTasksToJsonFile()
//	return
//}

//func (s *MysqlDbUtils) UpdateTask(t *Tunnel) error {
//	s.JsonDb.Tasks.Store(t.Id, t)
//	s.JsonDb.StoreTasksToJsonFile()
//	return nil
//}
//
//func (s *MysqlDbUtils) DelTask(id int) error {
//	s.JsonDb.Tasks.Delete(id)
//	s.JsonDb.StoreTasksToJsonFile()
//	return nil
//}
//
////md5 password
//func (s *MysqlDbUtils) GetTaskByMd5Password(p string) (t *Tunnel) {
//	s.JsonDb.Tasks.Range(func(key, value interface{}) bool {
//		if crypt.Md5(value.(*Tunnel).Password) == p {
//			t = value.(*Tunnel)
//			return false
//		}
//		return true
//	})
//	return
//}
//
//func (s *MysqlDbUtils) GetTask(id int) (t *Tunnel, err error) {
//	if v, ok := s.JsonDb.Tasks.Load(id); ok {
//		t = v.(*Tunnel)
//		return
//	}
//	err = errors.New("not found")
//	return
//}
//
func (s *MysqlDbUtils) DelHost(id int64, clientId int64) error {
	tx := s.mysqlDb.Model(&Host{})
	if id > 0 {
		tx.Where("id=? ", id)
	}
	if clientId > 0 {
		tx.Where("client_id=?", clientId)
	}
	var hostList []*Host
	err := tx.Find(&hostList).Error

	err = tx.Update("status", -1).Error

	if err == nil {
		if hostList != nil && len(hostList) > 0 {
			for i := range hostList {
				cache.GetRedis().Del(cache.HOST_URLMATCH_KEY + hostList[i].Scheme + "_" + hostList[i].Location).Result()
			}
		}
	}
	return err
}

func (s *MysqlDbUtils) IsHostExist(h *Host) bool {
	var exist bool
	var count int64
	err := s.mysqlDb.Model(&h).Where("id!=? and host=? and location=? and (scheme='all' or scheme=?)", h.Id, h.Host, h.Location, h.Scheme).Count(&count).Error
	if err != nil {
		panic(err)
	}
	if h != nil && h.Id > 0 {
		exist = true
	}
	return exist
}

func (s *MysqlDbUtils) HasHost(h *Host, c *Client) bool {
	var has bool
	var count int64

	err := s.mysqlDb.Model(&h).Where("client_id=? and host=? and location=?", c.Id, h.Host, h.Location, h.Scheme).Count(&count).Error
	if err != nil {
		panic(err)
	}
	if h != nil && h.Id > 0 {
		has = true
	}
	return has
}

func (s *MysqlDbUtils) NewHost(t *Host) error {
	t.ClientId = t.Client.Id
	if t.Location == "" {
		t.Location = "/"
	}
	if s.IsHostExist(t) {
		return errors.New("host has exist")
	}
	t.Flow = new(Flow)
	t.ToJSONString()

	err := s.mysqlDb.Save(t).Error
	//s.JsonDb.Hosts.Store(t.Id, t)
	//s.JsonDb.StoreHostToJsonFile()
	if err == nil {
		cache.GetRedis().Del(cache.HOST_URLMATCH_KEY + t.Scheme + "_" + t.Location).Result()
	}
	return err
}

func (s *MysqlDbUtils) GetHostPage(start, length int, clientId int64, search string, sort string, order string) ([]*Host, int) {
	list := make([]*Host, 0)
	var count int64

	var cnt int
	tx := s.mysqlDb.Model(&Host{}).Where("status>-1")
	if clientId != 0 {
		tx.Where("client_id=?", clientId)
	}
	if search != "" {
		tx.Where("(client_id=? or host like ? or remark like ?)", common.GetIntNoErrByStr(search), "%"+search+"%", "%"+search+"%")
	}
	err := tx.Count(&count).Error
	if err != nil {
		panic(err)
	}
	if sort == "" {
		sort = "id "
	}
	err = tx.Order(sort + order).Offset(start).Limit(length).Find(&list).Error
	if err != nil {
		panic(err)
	}
	for _, host := range list {
		host.ToJSONObject()
		host.Client, err = s.GetClient(host.ClientId)
	}
	return list, cnt
}
func (s *MysqlDbUtils) UpdateHost(t *Host) error {
	//clients:=services.GetClients()
	//clients.Store(t.Id, t)
	t.ClientId = t.Client.Id
	err := s.mysqlDb.Save(t).Error
	if err == nil {
		cache.GetRedis().Del(cache.HOST_URLMATCH_KEY + t.Scheme + "_" + t.Location).Result()
	}
	return err
}

func (s *MysqlDbUtils) GetHost() []*Host {
	list := make([]*Host, 0)
	err := s.mysqlDb.Model(&Host{}).Where("status>-1").Find(&list).Error

	if err != nil {
		panic(err)
	}
	for _, host := range list {
		host.ToJSONObject()
		host.Client, err = s.GetClient(host.ClientId)
	}
	return list
}

func (s *MysqlDbUtils) DelClient(id int64) error {
	err := s.mysqlDb.Model(&Client{}).Where("id=?", id).Update("status", -1).Error
	if err == nil {
		cache.GetRedis().Del(cache.CLIENT_INFO_KEY + strconv.FormatInt(id, 10)).Result()

	}
	return err
}
func (s *MysqlDbUtils) OffLineClient(id int64) error {
	err := s.mysqlDb.Model(&Client{}).Where("id=?", id).Update("is_connect", 0).Error
	if err == nil {
		cache.GetRedis().Del(cache.CLIENT_INFO_KEY + strconv.FormatInt(id, 10)).Result()

	}
	return err
}

func (s *MysqlDbUtils) NewClient(c *Client) error {
	var isNotSet bool
	if c.WebUserName != "" && !s.VerifyUserName(c.WebUserName, c.Id) {
		return errors.New("web login username duplicate, please reset")
	}
reset:
	if c.VerifyKey == "" || isNotSet {
		isNotSet = true
		c.VerifyKey = crypt.GetRandomString(16)
	}
	if c.RateLimit == 0 {
		c.Rate = rate.NewRate(int64(2 << 23))
	} else if c.Rate == nil {
		c.Rate = rate.NewRate(int64(c.RateLimit * 1024))
	}
	c.Rate.Start()
	if !s.VerifyVkey(c.VerifyKey, c.Id) {
		if isNotSet {
			goto reset
		}
		return errors.New("Vkey duplicate, please reset")
	}

	if c.Flow == nil {
		c.Flow = new(Flow)
	}
	c.ToJSONString()

	err := s.mysqlDb.Save(&c).Error
	if err != nil {
		logs.Error("保存客户端信息失败", err)
	}
	return err

}

func (s *MysqlDbUtils) VerifyVkey(vkey string, id int64) (res bool) {
	res = true
	client := &Client{}
	var count int64

	err := s.mysqlDb.Model(&client).Where("verify_key = ? and id!=?", vkey, id).Count(&count).Error
	if err != nil {
		panic(err)
	}
	res = count <= 0
	return
}
func (s *MysqlDbUtils) VerifyUserName(username string, id int64) (res bool) {
	res = true
	client := &Client{}
	var count int64

	err := s.mysqlDb.Model(&client).Where("web_user_name = ? and id!=?", username, id).Count(&count).Error
	if err != nil {
		panic(err)
	}
	res = count <= 0
	return
}

func (s *MysqlDbUtils) UpdateClient(t *Client) error {
	//clients:=services.GetClients()
	//clients.Store(t.Id, t)
	t.ToJSONString()
	err := s.mysqlDb.Save(t).Error
	if err == nil {
		cache.GetRedis().Del(cache.CLIENT_INFO_KEY + strconv.FormatInt(t.Id, 10)).Result()

	}
	//if t.RateLimit == 0 {
	//	t.Rate = rate.NewRate(int64(2 << 23))
	//	t.Rate.Start()
	//}
	return err
}

func (s *MysqlDbUtils) UpdateClientAddress(id int64, addr string, ver string, serverIp string) error {

	err := s.mysqlDb.Model(&Client{}).Where("id=?", id).Update("addr", addr).Update("version", ver).Update("is_connect", true).Update("server_ip", serverIp).Error
	if err == nil {
		cache.GetRedis().Del(cache.CLIENT_INFO_KEY + strconv.FormatInt(id, 10)).Result()
	}
	return err
}

func (s *MysqlDbUtils) UpdateClientStatus(id int64, status int) error {

	err := s.mysqlDb.Model(&Client{}).Where("id=?", id).Update("status", status).Error
	if err == nil {
		cache.GetRedis().Del(cache.CLIENT_INFO_KEY + strconv.FormatInt(id, 10)).Result()
	}
	return err
}

func (s *MysqlDbUtils) IsPubClient(id int64) bool {
	client, err := s.GetClient(id)
	if err == nil {
		return client.NoDisplay
	}
	return false
}

func (s *MysqlDbUtils) GetClient(id int64) (c *Client, err error) {

	err = s.mysqlDb.Where("id=? and status!=-1", id).Find(&c).Error
	if err != nil {
		return
	}
	if c == nil {
		err = errors.New("未找到客户端")
		return
	}
	c.ToJSONObject()

	return
}
func (s *MysqlDbUtils) GetClientByCache(id int64) (c *Client, err error) {

	bytes, err := cache.GetRedis().Get(cache.CLIENT_INFO_KEY + strconv.FormatInt(id, 10)).Bytes()
	if err == nil {
		err = json.Unmarshal(bytes, &c)
		if err == nil && c != nil {
			return
		}
	}

	c, err = s.GetClient(id)
	if err == nil {
		var jsonbytes []byte
		jsonbytes, _ = json.Marshal(c)
		_, err = cache.GetRedis().Set(cache.CLIENT_INFO_KEY+strconv.FormatInt(id, 10), string(jsonbytes), cache.CLIENT_INFO_TIMEOUT).Result()

	} else {
		_, err = cache.GetRedis().Set(cache.CLIENT_INFO_KEY+strconv.FormatInt(id, 10), "{}", cache.CLIENT_INFO_ERROR_TIMEOUT).Result()

	}

	return
}
func (s *MysqlDbUtils) GetClientIdByVkey(vkey string) (client *Client, err error) {

	list := make([]*Client, 0)
	err = s.mysqlDb.Where("status!=-1 and verify_key=?", vkey).Find(&list).Error
	if err != nil {
		return
	}

	var exist bool
	for _, c := range list {
		if common.Getverifyval(c.VerifyKey) == vkey {
			client = c
			exist = true
			break
		}
	}

	if exist {
		return
	}
	err = errors.New("未找到客户端")
	return
}

//func (s *JsonDb) LoadClientFromDB() {
//	loadSyncMapFromFile(s.ClientFilePath, func(v string) {
//		post := new(Client)
//		if json.Unmarshal([]byte(v), &post) != nil {
//			return
//		}
//		if post.RateLimit > 0 {
//			post.Rate = rate.NewRate(int64(post.RateLimit * 1024))
//		} else {
//			post.Rate = rate.NewRate(int64(2 << 23))
//		}
//		post.Rate.Start()
//		post.NowConn = 0
//		s.Clients.Store(post.Id, post)
//		if post.Id > int64(s.ClientIncreaseId) {
//			s.ClientIncreaseId = int32(post.Id)
//		}
//	})
//}

func (s *MysqlDbUtils) GetHostById(id int64) (h *Host, err error) {
	err = s.mysqlDb.Where("id=?", id).Find(&h).Error
	if err != nil {
		panic(err)
	}
	if h != nil {
		h.ToJSONObject()
		if h.Client, err = s.GetClient(h.ClientId); err != nil {
			if err != nil {
				panic(err)
			}
		}
		return
	}
	err = errors.New("The host could not be parsed")
	return
}

//get key by host from x
func (s *MysqlDbUtils) GetInfoByHost(host string, r *http.Request) (h *Host, err error) {
	var hosts []*Host
	match_path := r.RequestURI[:strings.Index(r.RequestURI[1:], "/")+1]
	hostJson, err := cache.GetRedis().Get(cache.HOST_URLMATCH_KEY + r.URL.Scheme + "_" + match_path).Bytes()
	if err == nil {
		err = json.Unmarshal(hostJson, &h)
		if h != nil && err == nil {
			if h.Id <= 0 {
				err = errors.New("没有匹配到url" + match_path)
			}
			if h.Client, err = s.GetClientByCache(h.ClientId); err != nil {
				err = errors.New("client下线或不存在了" + strconv.FormatInt(h.ClientId, 10))
				h = nil
			}
			return
		}
	}

	list := make([]*Host, 0)

	err = s.mysqlDb.Where("status=1 and is_close=? and (scheme='all' or scheme=?) and location = ?", false, r.URL.Scheme, match_path).Find(&list).Error
	//err=s.mysqlDb.Where("status=1 and is_close=? and (scheme='all' or scheme=?) ",false,r.URL.Scheme).Find(&list).Error
	if err != nil {
		return
	}
	//Handling Ported Access
	host = common.GetIpByAddr(host)
	for _, v := range list {
		tmpHost := v.Host
		if strings.Contains(tmpHost, "*") {
			tmpHost = strings.Replace(tmpHost, "*", "", -1)
			if strings.Contains(host, tmpHost) {
				hosts = append(hosts, v)
			}
		} else if v.Host == host {
			hosts = append(hosts, v)
		}
	}

	for _, v := range hosts {
		//If not set, default matches all
		if v.Location == "" {
			v.Location = "/"
		}
		if strings.Index(r.RequestURI, v.Location) == 0 {
			if h == nil || (len(v.Location) > len(h.Location)) {
				h = v
			}
		}
	}
	if h != nil {
		h.ToJSONObject()
		jsonbytes, _ := json.Marshal(h)
		cache.GetRedis().Set(cache.HOST_URLMATCH_KEY+r.URL.Scheme+"_"+match_path, string(jsonbytes), cache.HOST_URLMATCH_TIMEOUT)

		if h.Client, err = s.GetClientByCache(h.ClientId); err != nil || h.Client == nil || h.Client.Status != 1 {
			err = errors.New("client下线或不存在了" + strconv.FormatInt(h.ClientId, 10))
			h = nil
		}
		return
	}
	err = errors.New("The host could not be parsed")
	cache.GetRedis().Set(cache.HOST_URLMATCH_KEY+r.URL.Scheme+"_"+match_path, "{}", cache.HOST_URLMATCH_TIMEOUT)
	return
}
