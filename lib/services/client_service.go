package services

import (
	"errors"
	"dewfn.com/nps/lib/file"
	"dewfn.com/nps/lib/rate"
	"sync"
)

type ClientService struct {
}

var clients sync.Map

func GetClients() *sync.Map {
	return &clients
}
func (s *ClientService) LoadClientFromDB(list []*file.Client) {
	//list:= file.GetMysqlDb().GetClientList()
	for _, client := range list {

		if client.RateLimit > 0 {
			client.Rate = rate.NewRate(int64(client.RateLimit * 1024))
		} else {
			client.Rate = rate.NewRate(int64(2 << 23))
		}
		client.Rate.Start()
		client.NowConn = 0
		clients.Store(client.Id, client)
	}
}

func (s *ClientService) GetClient(id int64) (c *file.Client, err error) {
	if v, ok := clients.Load(id); ok {
		c = v.(*file.Client)
		return
	}
	err = errors.New("未找到客户端")
	return
}

func (s *ClientService) FullClientRealRateFlow(client *file.Client, isStart bool) (err error) {
	if isStart {
		if client.RateLimit > 0 {
			client.Rate = rate.NewRate(int64(client.RateLimit * 1024))
		} else {
			client.Rate = rate.NewRate(int64(2 << 23))
		}
		client.Rate.Start()
		client.NowConn = 0
		clients.Store(client.Id, client)
	} else {
		if v, ok := clients.Load(client.Id); ok {
			nowc := v.(*file.Client)
			client.Rate = nowc.Rate
			client.Flow = nowc.Flow
			return
		}
		err = errors.New("未找到客户端")
	}

	return
}
