package global

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"sync"
)

type Application struct {
	dbs     map[string]*gorm.DB
	engine  http.Handler
	mux     sync.RWMutex
	handler map[string][]func(r *gin.RouterGroup, hand ...*gin.HandlerFunc)
	routers []Router
}

type Router struct {
	HttpMethod, RelativePath, Handler string
}

type Routers struct {
	List []Router
}

var App = &Application{
	dbs:     make(map[string]*gorm.DB),
	handler: make(map[string][]func(r *gin.RouterGroup, hand ...*gin.HandlerFunc)),
	routers: make([]Router, 0),
}

// SetDb 设置对应key的db
func (e *Application) SetDb(key string, db *gorm.DB) {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.dbs[key] = db
}

// GetDb 获取所有map里的db数据
func (e *Application) GetDb() map[string]*gorm.DB {
	e.mux.Lock()
	defer e.mux.Unlock()
	return e.dbs
}

// GetDbByKey 根据key获取db
func (e *Application) GetDbByKey(key string) *gorm.DB {
	e.mux.Lock()
	defer e.mux.Unlock()
	if db, ok := e.dbs["*"]; ok {
		return db
	}
	return e.dbs[key]
}

// SetEngine 设置路由引擎
func (e *Application) SetEngine(engine http.Handler) {
	e.engine = engine
}

// GetEngine 获取路由引擎
func (e *Application) GetEngine() http.Handler {
	return e.engine
}

// GetRouter 获取路由表
func (e *Application) GetRouter() []Router {
	return e.setRouter()
}

// setRouter 设置路由表
func (e *Application) setRouter() []Router {
	switch e.engine.(type) {
	case *gin.Engine:
		routers := e.engine.(*gin.Engine).Routes()
		for _, router := range routers {
			e.routers = append(e.routers, Router{RelativePath: router.Path, Handler: router.Handler, HttpMethod: router.Method})
		}
	}
	return e.routers
}

// NewConfig 默认值
func NewConfig() *Application {
	return &Application{
		dbs:     make(map[string]*gorm.DB),
		handler: make(map[string][]func(r *gin.RouterGroup, hand ...*gin.HandlerFunc)),
		routers: make([]Router, 0),
	}
}

func (e *Application) SetHandler(key string, routerGroup func(r *gin.RouterGroup, hand ...*gin.HandlerFunc)) {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.handler[key] = append(e.handler[key], routerGroup)
}

func (e *Application) GetHandler() map[string][]func(r *gin.RouterGroup, hand ...*gin.HandlerFunc) {
	e.mux.Lock()
	defer e.mux.Unlock()
	return e.handler
}

func (e *Application) GetHandlerPrefix(key string) []func(r *gin.RouterGroup, hand ...*gin.HandlerFunc) {
	e.mux.Lock()
	defer e.mux.Unlock()
	return e.handler[key]
}
