package controllers

import (
	"encoding/hex"
	"dewfn.com/nps/lib/services"
	"dewfn.com/nps/server"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"dewfn.com/nps/lib/crypt"
)

type AuthController struct {
	beego.Controller
}

func (s *AuthController) GetAuthKey() {
	m := make(map[string]interface{})
	defer func() {
		s.Data["json"] = m
		s.ServeJSON()
	}()
	if cryptKey := beego.AppConfig.String("auth_crypt_key"); len(cryptKey) != 16 {
		m["status"] = 0
		return
	} else {
		b, err := crypt.AesEncrypt([]byte(beego.AppConfig.String("auth_key")), []byte(cryptKey))
		if err != nil {
			m["status"] = 0
			return
		}
		m["status"] = 1
		m["crypt_auth_key"] = hex.EncodeToString(b)
		m["crypt_type"] = "aes cbc"
		return
	}
}

func (s *AuthController) GetTime() {
	m := make(map[string]interface{})
	m["time"] = time.Now().Unix()
	s.Data["json"] = m
	s.ServeJSON()
}

//删除客户端
func (s *AuthController) RemoteAction() {
	id := s.GetInt64NoErr("id")
	actions := strings.Split(s.GetString("actions"), ",")
	for _, a := range actions {
		if a == "DelTunnelAndHostByClientId" {
			server.DelTunnelAndHostByClientId(id, false)
		} else if a == "DelClientConnect" {
			server.DelClientConnect(id)
		} else if a == "Delete" {
			clients := services.GetClients()
			clients.Delete(id)
		}
	}
	s.AjaxOk("delete success", nil)
}
func (s *AuthController) GetInt64NoErr(key string, def ...int64) int64 {
	strv := s.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0]
	}
	val, _ := strconv.ParseInt(strv, 0, 64)
	return val
}
func (s *AuthController) AjaxOk(str string, data interface{}) {
	s.Data["json"] = ajax(str, 1, data)
	s.ServeJSON()
	s.StopRun()
}
