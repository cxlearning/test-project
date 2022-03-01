package geecache

// 根据 key 获得 对应 group 的 PeerGetter（从group中获得val）
type PeerPicker interface {
	PickPeer(key string)(peer PeerGetter, ok bool)
}

type PeerGetter interface {
	Get(group string, key string)([]byte, error)
}