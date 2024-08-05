package services

import (
	"errors"
	"dewfn.com/nps/lib/file"
	"dewfn.com/nps/lib/global"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

type UserInfoService struct {
}

func (s UserInfoService) GetUserByNamePwd(userName string, password string) (*file.UserInfo, error) {
	if strings.TrimSpace(userName) == "" || strings.TrimSpace(password) == "" {
		return nil, errors.New("请输入用户名或密码.")
	}
	orm := global.App.GetDb()["*"]
	user := &file.UserInfo{}

	err := orm.Raw("select * from userinfo where user_name=? and status=1", userName).Scan(user).Error
	if err != nil {
		logs.Error("用户或密码错误.", err)
		return nil, err
	}
	if user.Password != password {
		return nil, errors.New("用户或密码错误.")
	}
	return user, nil
}
