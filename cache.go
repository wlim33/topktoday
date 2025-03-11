package main

import (
	"log"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/gofrs/uuid/v5"
)

func initCache() *lru.TwoQueueCache[uuid.UUID, *LeaderboardResponse] {
	cache, err := lru.New2Q[uuid.UUID, *LeaderboardResponse](128)

	if err != nil {
		log.Fatalf("Failed to initialize cache: %s", err)
	}
	return cache
}
