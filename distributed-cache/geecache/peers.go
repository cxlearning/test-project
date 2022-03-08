package geecache

// 根据 key 获得对应节点的PeerGetter
type PeerPicker interface {
	PickPeer(key string)(peer PeerGetter, ok bool)
}

type PeerGetter interface {
	Get(group string, key string)([]byte, error)
}