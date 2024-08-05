package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"dewfn.com/nps/lib/file"
	"dewfn.com/nps/server/session"
)

type ApiController struct {
	beego.Controller
}

func (s *ApiController) GetLiveAllServer() {
	serverList := session.GetLiveServer()
	if serverList == nil {
		s.AjaxErr("获取可用服务异常，请重试或联系管理员")
		return
	}
	s.AjaxOk("成功", serverList)
}
func (s *ApiController) GetLiveServer() {
	vkey := s.GetString("vkey")
	client, err := file.GetMysqlDb().GetClientIdByVkey(vkey)
	if err != nil {
		s.AjaxErr(fmt.Sprintf("没有找到客户端，%s 的配置", vkey))
		return
	}
	if client.Status != 1 {
		s.AjaxErr(fmt.Sprintf("客户端未启用，%s", vkey))
		return
	}
	_, serverHost := session.DistributionLiveServer(client.ServerIp)
	if serverHost == "" {
		s.AjaxErr("获取可用服务异常，请重试或联系管理员")
		return
	}

	s.AjaxOk("成功", serverHost)
}
func (s *ApiController) ok() {
	s.AjaxOk("成功", nil)
}

//ajax正确返回
func (s *ApiController) AjaxOk(str string, data interface{}) {
	s.Data["json"] = ajax(str, 1, data)
	s.ServeJSON()
	s.StopRun()
}

//ajax错误返回
func (s *ApiController) AjaxErr(str string) {
	s.Data["json"] = ajax(str, 0, nil)
	s.ServeJSON()
	s.StopRun()
}
