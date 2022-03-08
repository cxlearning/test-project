package geecache

import (
	"errors"
	"github.com/test-project/distributed-cache/singleflight"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

/*
lru --> cache --> group
PeerPicker  --> group (PeerPicker接口 使用 httppool 实例)


httppool 拥有根据 keys 查找节点  并从节点 获得 val 的能力， 还具有提供 http 服务的能力

根据 keys 查找节点：
	consistenthash --> httppool
	PickPeer   --> 被实现 HTTPPool
节点 获得 val 的能力：
	httpGetter     --> httppool
	PeerGetter --> 被实现  httpGetter


*/
type Group struct {
	name      string
	getter    Getter // 未命中， 从数据源中拿数据
	mainCache cache
	peers     PeerPicker  // HTTPPool注入
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex // 锁 groups
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.Lock()
	defer mu.Unlock()

	g := groups[name]
	return g
}

//------ 分布式 节点的方法

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) load(key string) (value ByteView, err error) {

	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if val, err := g.getFromPeer(peer, key); err != nil {
					return val, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// 根据key获取， 会选择节点，如果没有选择的节点 或 选到自己，则从本地获取
//func (g *Group) load(key string) (value ByteView, err error) {
//
//	if g.peers != nil {
//		if getter, ok := g.peers.PickPeer(key); ok {
//			if value, err =  g.getFromGetter(getter, key); err != nil {
//				return value, nil
//			}
//			log.Println("[GeeCache] Failed to get from peer", err)
//		}
//	}
//	// 从数据源获取
//	return g.getLocally(key)
//}

// 调用 getter 获得数据
func (g *Group) getFromGetter(getter PeerGetter, key string) (ByteView, error) {
	bytes, err := getter.Get(g.name, key) //一个节点可能在多个group中，所以需要groupName
	if err != nil {
		return ByteView{}, nil
	}
	return ByteView{b: bytes}, nil
}

// -------------   下面是 单节点的方法
// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {

	if key == "" {
		return ByteView{}, errors.New("key is empty")
	}

	// 从缓存中拿数据
	if val, ok := g.mainCache.get(key); ok {
		return val, nil
	}
	// 从其他节点获取
	return g.load(key)

}

//func (g *Group) load(key string) (value ByteView, err error) {
//	return g.getLocally(key)
//}

// 调用getter 从数据源中 找数据, 并保存到 缓存中
func (g *Group) getLocally(key string) (ByteView, error) {

	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneByte(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
