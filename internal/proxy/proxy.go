package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

// Proxy 按 Host 转发的反向代理
type Proxy struct {
	mu     sync.RWMutex
	routes map[string]*httputil.ReverseProxy // host -> proxy
}

// New 创建空路由表
func New() *Proxy {
	return &Proxy{routes: make(map[string]*httputil.ReverseProxy)}
}

// SetRoutes 全量替换路由表；backendMap 为 host -> backend URL（如 http://127.0.0.1:3000）
func (p *Proxy) SetRoutes(backendMap map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routes = make(map[string]*httputil.ReverseProxy)
	for host, backend := range backendMap {
		target, err := url.Parse(backend)
		if err != nil {
			log.Printf("invalid backend url for host=%q backend=%q: %v", host, backend, err)
			continue
		}
		p.routes[host] = httputil.NewSingleHostReverseProxy(target)
	}
}

// ServeHTTP 根据 Host 选择后端并转发，无匹配则 404
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	proxy, ok := p.routes[r.Host]
	p.mu.RUnlock()
	if !ok || proxy == nil {
		http.Error(w, "no upstream for host", http.StatusNotFound)
		return
	}
	proxy.ServeHTTP(w, r)
}
