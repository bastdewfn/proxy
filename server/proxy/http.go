package proxy

import (
	"bufio"
	"crypto/tls"
	"github.com/astaxie/beego"
	"io"
	"dewfn.com/nps/lib/services"
	"dewfn.com/nps/server/session"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"dewfn.com/nps/bridge"
	"dewfn.com/nps/lib/cache"
	"dewfn.com/nps/lib/common"
	"dewfn.com/nps/lib/conn"
	"dewfn.com/nps/lib/file"
	"dewfn.com/nps/server/connection"
)

type httpServer struct {
	BaseServer
	httpPort      int
	httpsPort     int
	httpServer    *http.Server
	httpsServer   *http.Server
	httpsListener net.Listener
	useCache      bool
	addOrigin     bool
	cache         *cache.Cache
	cacheLen      int
}

var httpForwardClient = &http.Client{
	Timeout: time.Second * 120,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   120 * time.Second,
			KeepAlive: 120 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   2,
		MaxIdleConns:          200,
		MaxConnsPerHost:       1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   100 * time.Second,
		ExpectContinueTimeout: 120 * time.Second,
	},
}

func NewHttp(bridge *bridge.Bridge, c *file.Tunnel, httpPort, httpsPort int, useCache bool, cacheLen int, addOrigin bool) *httpServer {
	httpServer := &httpServer{
		BaseServer: BaseServer{
			task:   c,
			bridge: bridge,
			Mutex:  sync.Mutex{},
		},
		httpPort:  httpPort,
		httpsPort: httpsPort,
		useCache:  useCache,
		cacheLen:  cacheLen,
		addOrigin: addOrigin,
	}
	if useCache {
		httpServer.cache = cache.New(cacheLen)
	}
	return httpServer
}

func (s *httpServer) Start() error {
	var err error
	if s.errorContent, err = common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "web", "static", "page", "error.html")); err != nil {
		s.errorContent = []byte("nps 404")
	}
	if s.httpPort > 0 {
		s.httpServer = s.NewServer(s.httpPort, "http")
		go func() {
			l, err := connection.GetHttpListener()
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
			err = s.httpServer.Serve(l)
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
		}()
	}
	if s.httpsPort > 0 {
		s.httpsServer = s.NewServer(s.httpsPort, "https")
		go func() {
			s.httpsListener, err = connection.GetHttpsListener()
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
			logs.Error(NewHttpsServer(s.httpsListener, s.bridge, s.useCache, s.cacheLen).Start())
		}()
	}
	return nil
}

func (s *httpServer) Close() error {
	if s.httpsListener != nil {
		s.httpsListener.Close()
	}
	if s.httpsServer != nil {
		s.httpsServer.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	return nil
}

func (s *httpServer) handleTunneling(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	c, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	s.handleHttp(conn.NewConn(c), r)
}

func (s *httpServer) handleHttp(c *conn.Conn, r *http.Request) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		connClient io.ReadWriteCloser
		scheme     = r.URL.Scheme
		lk         *conn.Link
		targetAddr string
		lenConn    *conn.LenConn
		isReset    bool
		//wg         sync.WaitGroup
	)

	defer func() {
		if connClient != nil {
			connClient.Close()
		} else {
			s.writeConnFail(c.Conn)
		}
		c.Close()
	}()
reset:
	if isReset {
		host.Client.AddConn()
	}
	if host, err = file.GetMysqlDb().GetInfoByHost(r.Host, r); err != nil {
		logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
		return
	}
	clientSesssion, err := session.GetServerByClient(host.ClientId)
	if clientSesssion == nil {
		logs.Error("客户端 %d 没有启动或没有连接上!", host.ClientId)
		return
	}
	//是否为自服务器注册的客户端，如果不是 则转发
	if !session.IsLocalServer(clientSesssion.ServerIp) {
		newUrl := "http://" + clientSesssion.ServerIp + ":" + beego.AppConfig.String("http_proxy_port") + r.RequestURI

		newR, err := http.NewRequest(r.Method, newUrl, r.Body)
		if err != nil {
			logs.Error("转调服务HTTP异常"+clientSesssion.ServerIp, err)
			return
		}
		newR.Header = r.Header
		rsp, err := httpForwardClient.Do(newR)
		if err != nil {
			logs.Error("转调服务HTTP异常"+clientSesssion.ServerIp, err)
			return
		}

		lenConn := conn.NewLenConn(c)
		defer rsp.Body.Close()
		if err := rsp.Write(lenConn); err != nil {
			logs.Error(err)
			return
		}
		//转发服务
		return
	}

	(&services.ClientService{}).FullClientRealRateFlow(host.Client, false)

	if err := s.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		return
	}
	if !isReset {
		defer host.Client.AddConn()
	}
	if err = s.auth(r, c, host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
		logs.Warn("auth error", err, r.RemoteAddr)
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		logs.Warn(err.Error())
		return
	}
	lk = conn.NewLink("http", targetAddr, host.Client.Cnf.Crypt, host.Client.Cnf.Compress, r.RemoteAddr, host.Target.LocalProxy)
	if target, err = s.bridge.SendLinkInfo(host.Client.Id, lk, nil); err != nil {
		logs.Notice("connect to target %s error %s", lk.Host, err)
		return
	}
	connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
	//read from inc-client
	go func() {
		//wg.Add(1)
		isReset = false
		defer connClient.Close()
		defer func() {
			//wg.Done()
			if !isReset {
				c.Close()
			}
		}()
		for {
			if resp, err := http.ReadResponse(bufio.NewReader(connClient), r); err != nil || resp == nil || r == nil {
				logs.Info(err)
				// if there got broken pipe, http.ReadResponse will get a nil
				return
			} else {
				//if the cache is start and the response is in the extension,store the response to the cache list
				if s.useCache && r.URL != nil && strings.Contains(r.URL.Path, ".") {
					b, err := httputil.DumpResponse(resp, true)
					if err != nil {
						return
					}
					c.Write(b)
					host.Flow.Add(0, int64(len(b)))
					s.cache.Add(filepath.Join(host.Host, r.URL.Path), b)
				} else {
					lenConn := conn.NewLenConn(c)
					if err := resp.Write(lenConn); err != nil {
						logs.Error(err)
						return
					}
					host.Flow.Add(0, int64(lenConn.Len))
				}
			}
		}
	}()

	for {
		//if the cache start and the request is in the cache list, return the cache
		if s.useCache {
			if v, ok := s.cache.Get(filepath.Join(host.Host, r.URL.Path)); ok {
				n, err := c.Write(v.([]byte))
				if err != nil {
					logs.Error(err)
					break
				}
				logs.Trace("%s request, method %s, host %s, url %s, remote address %s, return cache", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String())
				host.Flow.Add(0, int64(n))
				//if return cache and does not create a new conn with client and Connection is not set or close, close the connection.
				if strings.ToLower(r.Header.Get("Connection")) == "close" || strings.ToLower(r.Header.Get("Connection")) == "" {
					break
				}
				goto readReq
			}
		}

		//change the host and header and set proxy setting
		common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, c.Conn.RemoteAddr().String(), s.addOrigin, host.Location, host.ReplaceLocation)
		logs.Trace("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String(), lk.Host)
		//write
		lenConn = conn.NewLenConn(connClient)
		if err := r.Write(lenConn); err != nil {
			logs.Error(err)
			break
		}
		host.Flow.Add(int64(lenConn.Len), 0)

	readReq:
		//read req from connection
		//c.SetDeadline(time.Now().Add(time.Second * 30))
		if r, err = http.ReadRequest(bufio.NewReader(c)); err != nil {
			logs.Info(err)
			break
		}
		r.URL.Scheme = scheme
		//What happened ，Why one character less???
		r.Method = resetReqMethod(r.Method)
		if hostTmp, err := file.GetMysqlDb().GetInfoByHost(r.Host, r); err != nil {
			logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
			break
		} else if host.Id != hostTmp.Id {
			(&services.ClientService{}).FullClientRealRateFlow(hostTmp.Client, false)
			host = hostTmp
			isReset = true
			connClient.Close()
			goto reset
		}
	}
	//wg.Wait()
}

func resetReqMethod(method string) string {
	if method == "ET" {
		return "GET"
	}
	if method == "OST" {
		return "POST"
	}
	return method
}

func (s *httpServer) NewServer(port int, scheme string) *http.Server {
	return &http.Server{
		Addr: ":" + strconv.Itoa(port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = scheme
			s.handleTunneling(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
}
