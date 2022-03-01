package consistenthash

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash           //hash算法，可以自定义
	replicas int            // 虚拟节点个数
	keys     []int          // Sorted
	hashMap  map[int]string // 存储 虚拟节点 与 真实节点 的关系
}

func(m *Map) PrintKeys()  {
	fmt.Println(m.keys)
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {

	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
// 添加节点
func (m *Map) Add(keys ...string) {

	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hashVal := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hashVal)
			m.hashMap[hashVal] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
// 得到key 应该访问的 节点
func (m *Map) Get(key string) string {

	hashVal := int(m.hash([]byte(key)))

	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hashVal
	})
	return m.hashMap[m.keys[idx % len(m.keys)]]
}
