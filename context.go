package boomer

import (
	"encoding/json"
	"sync"
)

const (
	_shardCount = 32
)

// A "thread" safe map of type string:Anything.
// To avoid lock bottlenecks this map is dived to several (_shardCount) map shards.
type Context []*ContextShared

// A "thread" safe string to anything map.
type ContextShared struct {
	items        map[string]interface{}
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

func NewContext() Context {
	m := make(Context, _shardCount)
	for i := 0; i < _shardCount; i++ {
		m[i] = &ContextShared{items: make(map[string]interface{})}
	}
	return m
}

// GetShard returns shard under given key
func (m Context) GetShard(key string) *ContextShared {
	return m[uint(fnv32(key))%uint(_shardCount)]
}

func (m Context) MSet(data map[string]interface{}) {
	for key, value := range data {
		shard := m.GetShard(key)
		shard.Lock()
		shard.items[key] = value
		shard.Unlock()
	}
}

// Sets the given value under the specified key.
func (m Context) Set(key string, value interface{}) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// Callback to return new element to be inserted into the map
// It is called while lock is held, therefore it MUST NOT
// try to access other keys in same map, as it can lead to deadlock since
// Go sync.RWLock is not reentrant
type UpsertCb func(exist bool, valueInMap interface{}, newValue interface{}) interface{}

// Insert or Update - updates existing element or inserts a new one using UpsertCb
func (m Context) Upsert(key string, value interface{}, cb UpsertCb) (res interface{}) {
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	res = cb(ok, v, value)
	shard.items[key] = res
	shard.Unlock()
	return res
}

// Sets the given value under the specified key if no value was associated with it.
func (m Context) SetIfAbsent(key string, value interface{}) bool {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok
}

// Get retrieves an element from map under given key.
func (m Context) Get(key string) (interface{}, bool) {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// Get item from shard.
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

func (m Context) GetString(key string) (value string, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(string); !ok {
		return
	}
	return
}

func (m Context) GetInt(key string) (value int, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(int); !ok {
		return
	}
	return
}

func (m Context) GetInt8(key string) (value int8, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(int8); !ok {
		return
	}
	return
}

func (m Context) GetInt16(key string) (value int16, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(int16); !ok {
		return
	}
	return
}

func (m Context) GetInt32(key string) (value int32, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(int32); !ok {
		return
	}
	return
}

func (m Context) GetInt64(key string) (value int64, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(int64); !ok {
		return
	}
	return
}

func (m Context) GetUint(key string) (value uint, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(uint); !ok {
		return
	}
	return
}

func (m Context) GetUint8(key string) (value uint8, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(uint8); !ok {
		return
	}
	return
}

func (m Context) GetUint16(key string) (value uint16, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(uint16); !ok {
		return
	}
	return
}

func (m Context) GetUint32(key string) (value uint32, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(uint32); !ok {
		return
	}
	return
}

func (m Context) GetUint64(key string) (value uint64, ok bool) {
	var v interface{}
	if v, ok = m.Get(key); !ok {
		return
	}

	if value, ok = v.(uint64); !ok {
		return
	}
	return
}

// Count returns the number of elements within the map.
func (m Context) Count() int {
	count := 0
	for i := 0; i < _shardCount; i++ {
		shard := m[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m Context) Has(key string) bool {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// See if element is within shard.
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// Remove removes an element from the map.
func (m Context) Remove(key string) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// RemoveCb is a callback executed in a map.RemoveCb() call, while Lock is held
// If returns true, the element will be removed from the map
type RemoveCb func(key string, v interface{}, exists bool) bool

// RemoveCb locks the shard containing the key, retrieves its current value and calls the callback with those params
// If callback returns true and element exists, it will remove it from the map
// Returns the value returned by the callback (even if element was not present in the map)
func (m Context) RemoveCb(key string, cb RemoveCb) bool {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	remove := cb(key, v, ok)
	if remove && ok {
		delete(shard.items, key)
	}
	shard.Unlock()
	return remove
}

// Pop removes an element from the map and returns it
func (m Context) Pop(key string) (v interface{}, exists bool) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	v, exists = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return v, exists
}

func (m Context) PopString(key string) (value string, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(string); !ok {
		return
	}
	return
}

func (m Context) PopInt(key string) (value int, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(int); !ok {
		return
	}
	return
}

func (m Context) PopInt8(key string) (value int8, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(int8); !ok {
		return
	}
	return
}

func (m Context) PopInt16(key string) (value int16, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(int16); !ok {
		return
	}
	return
}

func (m Context) PopInt32(key string) (value int32, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(int32); !ok {
		return
	}
	return
}

func (m Context) PopInt64(key string) (value int64, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(int64); !ok {
		return
	}
	return
}

func (m Context) PopUint(key string) (value uint, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(uint); !ok {
		return
	}
	return
}

func (m Context) PopUint8(key string) (value uint8, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(uint8); !ok {
		return
	}
	return
}

func (m Context) PopUint16(key string) (value uint16, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(uint16); !ok {
		return
	}
	return
}

func (m Context) PopUint32(key string) (value uint32, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(uint32); !ok {
		return
	}
	return
}

func (m Context) PopUint64(key string) (value uint64, ok bool) {
	var v interface{}
	if v, ok = m.Pop(key); !ok {
		return
	}

	if value, ok = v.(uint64); !ok {
		return
	}
	return
}

// IsEmpty checks if map is empty.
func (m Context) IsEmpty() bool {
	return m.Count() == 0
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple struct {
	Key string
	Val interface{}
}

// Iter returns an iterator which could be used in a for range loop.
//
// Deprecated: using IterBuffered() will get a better performence
func (m Context) Iter() <-chan Tuple {
	chans := snapshot(m)
	ch := make(chan Tuple)
	go fanIn(chans, ch)
	return ch
}

// IterBuffered returns a buffered iterator which could be used in a for range loop.
func (m Context) IterBuffered() <-chan Tuple {
	chans := snapshot(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan Tuple, total)
	go fanIn(chans, ch)
	return ch
}

// Returns a array of channels that contains elements in each shard,
// which likely takes a snapshot of `m`.
// It returns once the size of each buffered channel is determined,
// before all the channels are populated using goroutines.
func snapshot(m Context) (chans []chan Tuple) {
	chans = make([]chan Tuple, _shardCount)
	wg := sync.WaitGroup{}
	wg.Add(_shardCount)
	// Foreach shard.
	for index, shard := range m {
		go func(index int, shard *ContextShared) {
			// Foreach key, value pair.
			shard.RLock()
			chans[index] = make(chan Tuple, len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chans[index] <- Tuple{key, val}
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanIn reads elements from channels `chans` into channel `out`
func fanIn(chans []chan Tuple, out chan Tuple) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan Tuple) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// Items returns all items as map[string]interface{}
func (m Context) Items() map[string]interface{} {
	tmp := make(map[string]interface{})

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
}

// Iterator callback,called for every key,value found in
// maps. RLock is held for all calls for a given shard
// therefore callback sess consistent view of a shard,
// but not across the shards
type IterCb func(key string, v interface{})

// Callback based iterator, cheapest way to read
// all elements in a map.
func (m Context) IterCb(fn IterCb) {
	for idx := range m {
		shard := (m)[idx]
		shard.RLock()
		for key, value := range shard.items {
			fn(key, value)
		}
		shard.RUnlock()
	}
}

// Keys returns all keys as []string
func (m Context) Keys() []string {
	count := m.Count()
	ch := make(chan string, count)
	go func() {
		// Foreach shard.
		wg := sync.WaitGroup{}
		wg.Add(_shardCount)
		for _, shard := range m {
			go func(shard *ContextShared) {
				// Foreach key, value pair.
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	// Generate keys
	keys := make([]string, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

//Reviles Context "private" variables to json marshal.
func (m Context) MarshalJSON() ([]byte, error) {
	// Create a temporary map, which will hold all item spread across shards.
	tmp := make(map[string]interface{})

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// Concurrent map uses Interface{} as its value, therefor JSON Unmarshal
// will probably won't know which to type to unmarshal into, in such case
// we'll end up with a value of type map[string]interface{}, In most cases this isn't
// out value type, this is why we've decided to remove this functionality.

// func (m *Context) UnmarshalJSON(b []byte) (err error) {
// 	// Reverse process of Marshal.

// 	tmp := make(map[string]interface{})

// 	// Unmarshal into a single map.
// 	if err := json.Unmarshal(b, &tmp); err != nil {
// 		return nil
// 	}

// 	// foreach key,value pair in temporary map insert into our concurrent map.
// 	for key, val := range tmp {
// 		m.Set(key, val)
// 	}
// 	return nil
// }
