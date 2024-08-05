package services

import (
	"errors"
	"dewfn.com/nps/lib/file"
	"sync"
)

type HostService struct {
}

var hosts sync.Map

func GetHosts() *sync.Map {
	return &hosts
}
func (s *HostService) LoadHostFromJsonFile(list []*file.Host) {
	clientService := &ClientService{}
	for _, post := range list {
		var err error

		if post.Client, err = clientService.GetClient(post.ClientId); err != nil {
			return
		}
		hosts.Store(post.Id, post)
	}
}

func (s *HostService) GetHost(id int64) (c *file.Host, err error) {
	if v, ok := clients.Load(id); ok {
		c = v.(*file.Host)
		return
	}
	err = errors.New("未找到Host")
	return
}
