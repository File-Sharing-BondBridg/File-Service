package services

import (
	"hash/fnv"
)

type Resolver struct {
	shardCount int
}

func NewResolver(shardCount int) *Resolver {
	return &Resolver{shardCount: shardCount}
}

func ResolveShard(userID string, shardCount int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return int(h.Sum32()) % shardCount
}
