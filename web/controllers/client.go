package controllers

import (
	"github.com/astaxie/beego"
	"dewfn.com/nps/lib/common"
	"dewfn.com/nps/lib/file"
	"dewfn.com/nps/server"
	"dewfn.com/nps/server/session"
)

type ClientController struct {
	BaseController
}

func (s *ClientController) List() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["menu"] = "client"
		s.SetInfo("client")
		s.display("client/list")
		return
	}
	start, length := s.GetAjaxParams()
	clientIdSession := s.GetSession("clientId")
	var clientId int64
	if clientIdSession == nil {
		clientId = 0
	} else {
		clientId = clientIdSession.(int64)
	}

	list, cnt := server.GetClientList(start, length, s.getEscapeString("search"), s.getEscapeString("sort"), s.getEscapeString("order"), clientId)
	cmd := make(map[string]interface{})
	ip := session.MyIp
	cmd["ip"] = ip
	cmd["bridgeType"] = beego.AppConfig.String("bridge_type")
	cmd["bridgePort"] = server.Bridge.TunnelPort
	s.AjaxTable(list, int(cnt), int(cnt), cmd)
}

//添加客户端
func (s *ClientController) Add() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["menu"] = "client"
		s.SetInfo("add client")
		s.display()
	} else {
		t := &file.Client{
			VerifyKey: s.getEscapeString("vkey"),
			Model: file.Model{
				Status: 1,
			},
			Remark: s.getEscapeString("remark"),
			Cnf: &file.Config{
				U:        s.getEscapeString("u"),
				P:        s.getEscapeString("p"),
				Compress: common.GetBoolByStr(s.getEscapeString("compress")),
				Crypt:    s.GetBoolNoErr("crypt"),
			},
			ConfigConnAllow: s.GetBoolNoErr("config_conn_allow"),
			RateLimit:       s.GetIntNoErr("rate_limit"),
			MaxConn:         s.GetIntNoErr("max_conn"),
			WebUserName:     s.getEscapeString("web_username"),
			WebPassword:     s.getEscapeString("web_password"),
			MaxTunnelNum:    s.GetIntNoErr("max_tunnel"),
			Flow: &file.Flow{
				ExportFlow: 0,
				InletFlow:  0,
				FlowLimit:  int64(s.GetIntNoErr("flow_limit")),
			},
		}

		if err := file.GetMysqlDb().NewClient(t); err != nil {
			s.AjaxErr(err.Error())
		}
		//clients := services.GetClients()
		//clients.Store(t.Id, t)
		s.AjaxOk("add success")
	}
}
func (s *ClientController) GetClient() {
	if s.Ctx.Request.Method == "POST" {
		id := s.GetInt64NoErr("id")
		data := make(map[string]interface{})

		if c, err := file.GetMysqlDb().GetClient(id); err != nil {
			data["code"] = 0
		} else {
			data["code"] = 1
			data["data"] = c

		}
		s.Data["json"] = data
		s.ServeJSON()
	}
}

//修改客户端
func (s *ClientController) Edit() {
	id := s.GetInt64NoErr("id")
	if s.Ctx.Request.Method == "GET" {
		s.Data["menu"] = "client"

		if c, err := file.GetMysqlDb().GetClient(id); err != nil {
			s.error()
		} else {
			s.Data["c"] = c
		}
		s.SetInfo("edit client")
		s.display()
	} else {
		if c, err := file.GetMysqlDb().GetClient(id); err != nil {
			s.error()
			s.AjaxErr("client ID not found")
			return
		} else {
			if s.getEscapeString("web_username") != "" {
				if s.getEscapeString("web_username") == beego.AppConfig.String("web_username") || !file.GetMysqlDb().VerifyUserName(s.getEscapeString("web_username"), c.Id) {
					s.AjaxErr("web login username duplicate, please reset")
					return
				}
			}
			if s.GetSession("isAdmin").(bool) {
				if !file.GetMysqlDb().VerifyVkey(s.getEscapeString("vkey"), c.Id) {
					s.AjaxErr("Vkey duplicate, please reset")
					return
				}
				c.VerifyKey = s.getEscapeString("vkey")
				c.Flow.FlowLimit = int64(s.GetIntNoErr("flow_limit"))
				c.RateLimit = s.GetIntNoErr("rate_limit")
				c.MaxConn = s.GetIntNoErr("max_conn")
				c.MaxTunnelNum = s.GetIntNoErr("max_tunnel")
			}
			c.Remark = s.getEscapeString("remark")
			c.Cnf.U = s.getEscapeString("u")
			c.Cnf.P = s.getEscapeString("p")
			c.Cnf.Compress = common.GetBoolByStr(s.getEscapeString("compress"))
			c.Cnf.Crypt = s.GetBoolNoErr("crypt")
			b, err := beego.AppConfig.Bool("allow_user_change_username")
			if s.GetSession("isAdmin").(bool) || (err == nil && b) {
				c.WebUserName = s.getEscapeString("web_username")
			}
			c.WebPassword = s.getEscapeString("web_password")
			c.ConfigConnAllow = s.GetBoolNoErr("config_conn_allow")
			//if c.Rate != nil {
			//	c.Rate.Stop()
			//}
			//if c.RateLimit > 0 {
			//	c.Rate = rate.NewRate(int64(c.RateLimit * 1024))
			//	c.Rate.Start()
			//} else {
			//	c.Rate = rate.NewRate(int64(2 << 23))
			//	c.Rate.Start()
			//}
			file.GetMysqlDb().UpdateClient(c)

			//services.GetClients().Store(c.Id, c)
		}
		s.AjaxOk("save success")
	}
}

//更改状态
func (s *ClientController) ChangeStatus() {
	id := s.GetInt64NoErr("id")
	var status int
	if s.GetBoolNoErr("status") {
		status = 1
	} else {
		status = 0
	}

	err := file.GetMysqlDb().UpdateClientStatus(id, status)
	if err == nil {
		//service := &services.ClientService{}
		//client, _ := service.GetClient(id)
		//client.Status = status
		if status == 0 {
			if !server.RemoteAction(id, "DelClientConnect") {
				server.DelClientConnect(id)
			}
		}
		s.AjaxOk("modified success")
	}
	s.AjaxErr("modified fail")
}

//删除客户端
func (s *ClientController) Del() {
	id := s.GetInt64NoErr("id")
	if err := file.GetMysqlDb().DelClient(id); err != nil {
		s.AjaxErr("delete error")
	}
	if !server.RemoteAction(id, "Delete,DelTunnelAndHostByClientId,DelClientConnect") {
		//clients := services.GetClients()
		//clients.Delete(id)
		server.DelTunnelAndHostByClientId(id, false)
		server.DelClientConnect(id)
	}
	s.AjaxOk("delete success")
}
