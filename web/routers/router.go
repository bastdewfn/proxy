package routers

import (
	"github.com/astaxie/beego"
	"dewfn.com/nps/web/controllers"
)

func Init() {
	web_base_url := beego.AppConfig.String("web_base_url")
	web_api_url := beego.AppConfig.String("web_api_url")

	ns := beego.NewNamespace(web_base_url,
		beego.NSRouter("/", &controllers.IndexController{}, "*:Index"),
		beego.NSAutoRouter(&controllers.IndexController{}),
		beego.NSAutoRouter(&controllers.LoginController{}),
		beego.NSAutoRouter(&controllers.ClientController{}),
		beego.NSAutoRouter(&controllers.AuthController{}),
	)
	nsapi := beego.NewNamespace(web_api_url,
		beego.NSAutoRouter(&controllers.ApiController{}),
	)
	beego.AddNamespace(ns, nsapi)

}
