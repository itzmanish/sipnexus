package sipnexus

import (
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
)

type ConsistentHash struct {
	circle       map[uint32]string
	sortedHashes []uint32
	virtualNodes int
	mu           sync.RWMutex
}

func NewConsistentHash(virtualNodes int) *ConsistentHash {
	return &ConsistentHash{
		circle:       make(map[uint32]string),
		virtualNodes: virtualNodes,
	}
}

func (c *ConsistentHash) Add(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := 0; i < c.virtualNodes; i++ {
		hash := c.hash(fmt.Sprintf("%s:%d", node, i))
		c.circle[hash] = node
		c.sortedHashes = append(c.sortedHashes, hash)
	}
	sort.Slice(c.sortedHashes, func(i, j int) bool {
		return c.sortedHashes[i] < c.sortedHashes[j]
	})
}

func (c *ConsistentHash) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.circle) == 0 {
		return ""
	}

	hash := c.hash(key)
	idx := sort.Search(len(c.sortedHashes), func(i int) bool {
		return c.sortedHashes[i] >= hash
	})

	if idx == len(c.sortedHashes) {
		idx = 0
	}

	return c.circle[c.sortedHashes[idx]]
}

func (c *ConsistentHash) hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
