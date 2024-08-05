package file

import (
	"encoding/json"
	"dewfn.com/nps/lib/version"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"dewfn.com/nps/lib/rate"
)

type Flow struct {
	ExportFlow int64
	InletFlow  int64
	FlowLimit  int64
	sync.RWMutex
}

func (s *Flow) Add(in, out int64) {
	s.Lock()
	defer s.Unlock()
	s.InletFlow += int64(in)
	s.ExportFlow += int64(out)
}

type Config struct {
	U        string
	P        string
	Compress bool
	Crypt    bool
}

type Client struct {
	Model
	Cnf             *Config    `gorm:"-"`
	CnfJson         string     `json:"-" gorm:"size:500;"`
	VerifyKey       string     `gorm:"size:100;"` //verify key
	Addr            string     `gorm:"size:100;"` //the ip of client
	ServerIp        string     `gorm:"size:50;"`  //the ip of server
	Remark          string     `gorm:"size:50;"`  //remark
	IsConnect       bool       //is the client connect
	RateLimit       int        //rate /kb
	Flow            *Flow      `gorm:"-"` //flow setting
	Rate            *rate.Rate `gorm:"-"` //rate limit
	FlowJson        string     `json:"-"  gorm:"size:500;"`
	RateJson        string     `json:"-" gorm:"size:500;"`
	ConnectTime     time.Time  `gorm:"-"`
	NoStore         bool       //no store to file
	NoDisplay       bool       //no display on web
	MaxConn         int        //the max connection num of client allow
	NowConn         int32      //the connection num of now
	WebUserName     string     `gorm:"size:50;"` //the username of web login
	WebPassword     string     `gorm:"size:50;"` //the password of web login
	ConfigConnAllow bool       //is allow connected by config file
	MaxTunnelNum    int
	Version         string `gorm:"size:10;"`
	sync.RWMutex
}

func NewClient(vKey string, noStore bool, noDisplay bool) *Client {
	return &Client{
		Model:     Model{Status: 1},
		Cnf:       new(Config),
		VerifyKey: vKey,
		Addr:      "",
		Remark:    "",
		IsConnect: false,
		RateLimit: 0,
		Flow:      new(Flow),
		Rate:      nil,
		NoStore:   noStore,
		RWMutex:   sync.RWMutex{},
		NoDisplay: noDisplay,
		Version:   version.GetVersion(),
	}
}
func (s *Client) ToJSONObject() error {
	err := json.Unmarshal([]byte(s.CnfJson), &s.Cnf)
	err = json.Unmarshal([]byte(s.RateJson), &s.Rate)
	err = json.Unmarshal([]byte(s.FlowJson), &s.Flow)
	return err
}
func (s *Client) ToJSONString() error {
	data, err := json.Marshal(s.Cnf)
	s.CnfJson = string(data)
	data, err = json.Marshal(s.Rate)
	s.RateJson = string(data)
	data, err = json.Marshal(s.Flow)
	s.FlowJson = string(data)
	return err
}

func (s *Client) CutConn() {
	atomic.AddInt32(&s.NowConn, 1)
}

func (s *Client) AddConn() {
	atomic.AddInt32(&s.NowConn, -1)
}

func (s *Client) GetConn() bool {
	if s.MaxConn == 0 || int(s.NowConn) < s.MaxConn {
		s.CutConn()
		return true
	}
	return false
}

func (s *Client) HasTunnel(t *Tunnel) (exist bool) {
	GetDb().JsonDb.Tasks.Range(func(key, value interface{}) bool {
		v := value.(*Tunnel)
		if v.Client.Id == s.Id && v.Port == t.Port && t.Port != 0 {
			exist = true
			return false
		}
		return true
	})
	return
}

func (s *Client) GetTunnelNum() (num int) {
	GetDb().JsonDb.Tasks.Range(func(key, value interface{}) bool {
		v := value.(*Tunnel)
		if v.Client.Id == s.Id {
			num++
		}
		return true
	})
	return
}

type Tunnel struct {
	Id           int
	Port         int
	ServerIp     string
	Mode         string
	Status       bool
	RunStatus    bool
	Client       *Client
	Ports        string
	Flow         *Flow
	Password     string
	Remark       string
	TargetAddr   string
	NoStore      bool
	LocalPath    string
	StripPre     string
	Target       *Target
	MultiAccount *MultiAccount
	Health
	sync.RWMutex
}

type Health struct {
	HealthCheckTimeout  int
	HealthMaxFail       int
	HealthCheckInterval int
	HealthNextTime      time.Time
	HealthMap           map[string]int
	HttpHealthUrl       string
	HealthRemoveArr     []string
	HealthCheckType     string
	HealthCheckTarget   string
	sync.RWMutex
}

type Host struct {
	Model
	Host            string `gorm:"size:100;"` //host
	HeaderChange    string `gorm:"size:500;"` //header change
	HostChange      string `gorm:"size:100;"` //host change
	Location        string `gorm:"size:100;"` //url router
	ReplaceLocation string `gorm:"size:100;"` //url router
	Remark          string `gorm:"size:50;"`  //remark
	Scheme          string `gorm:"size:50;"`  //http https all
	CertFilePath    string `gorm:"size:200;"`
	KeyFilePath     string `gorm:"size:200;"`
	FlowJson        string `json:"-" gorm:"size:500;"`
	TargetJson      string `json:"-" gorm:"size:500;"`
	NoStore         bool
	IsClose         bool
	Flow            *Flow   `gorm:"-"`
	Client          *Client `gorm:"-"`
	ClientId        int64
	Target          *Target `gorm:"-"` //目标
	Health          `json:"-" gorm:"-"`
	sync.RWMutex
}

func (s *Host) ToJSONObject() error {
	err := json.Unmarshal([]byte(s.TargetJson), &s.Target)
	err = json.Unmarshal([]byte(s.FlowJson), &s.Flow)
	return err
}
func (s *Host) ToJSONString() error {
	data, err := json.Marshal(s.Target)
	s.TargetJson = string(data)
	data, err = json.Marshal(s.Flow)
	s.FlowJson = string(data)
	return err
}

type Target struct {
	nowIndex   int
	TargetStr  string
	TargetArr  []string
	LocalProxy bool
	sync.RWMutex
}

type MultiAccount struct {
	AccountMap map[string]string // multi account and pwd
}

func (s *Target) GetRandomTarget() (string, error) {
	if s.TargetArr == nil {
		s.TargetArr = strings.Split(s.TargetStr, "\n")
	}
	if len(s.TargetArr) == 1 {
		return s.TargetArr[0], nil
	}
	if len(s.TargetArr) == 0 {
		return "", errors.New("all inward-bending targets are offline")
	}
	s.Lock()
	defer s.Unlock()
	if s.nowIndex >= len(s.TargetArr)-1 {
		s.nowIndex = -1
	}
	s.nowIndex++
	return s.TargetArr[s.nowIndex], nil
}

type UserInfo struct {
	Model

	UserName string `json:"userName" gorm:"size:50;"`  //the username of web login
	Password string `json:"passWord" gorm:"size:100;"` //the password of web login

}
