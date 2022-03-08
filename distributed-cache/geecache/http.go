package geecache

import (
	"fmt"
	"github.com/test-project/distributed-cache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)


/**
HTTPPool  可以理解为一个节点
	1 提供 http访问
	2 peers 和 httpGetters， 提供访问其他节点的能力
 */
// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	self        string // this peer's base URL, e.g. "https://example.net:8000", httppool 所在的节点
	basePath    string
	mu          sync.Mutex // 保证 peers httpGetters 线程 安全
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

//加入节点, peer 是 节点，例如 http:172.16.17.18:11088
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, _peer := range peers {
		p.httpGetters[_peer] = &httpGetter{baseURL:_peer + p.basePath }
	}
}

/**
实现接口 PeerPicker
根据 key 获得 节点的 PeerGetter
 */
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {

	p.mu.Lock()
	defer p.mu.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}

	return nil, false
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	strArr := strings.Split(r.URL.Path[len(p.basePath):], "/")
	if len(strArr) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	group, ok := groups[strArr[0]]
	if !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	val, err := group.Get(strArr[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(val.ByteSlice())
}

type httpGetter struct {
	baseURL string
}

/**
向 httpGetter对应节点 发送 http 请求获得val
一个节点可能在多个group中，所以需要groupName
*/
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v/%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	byte, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return byte, nil
}
