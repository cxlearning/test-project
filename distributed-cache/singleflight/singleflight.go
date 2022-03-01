package singleflight

import "sync"

/**
代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入。
*/
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

/**
singleflight 的主数据结构，管理不同 key 的请求(call)
*/
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

/**
Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
*/
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {

	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	if call, ok := g.m[key]; ok {
		g.mu.Unlock()
		call.wg.Wait()
		return call.val, call.err
	}

	call := &call{}
	call.wg.Add(1)
	g.m[key] = call
	g.mu.Unlock()

	call.val, call.err = fn()
	call.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return call.val, call.err
}
