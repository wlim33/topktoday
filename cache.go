package main

import (
	"log"

	lru "github.com/hashicorp/golang-lru/v2"
)

func initCache() *lru.TwoQueueCache[string, *LeaderboardResponse] {
	cache, err := lru.New2Q[string, *LeaderboardResponse](128)

	if err != nil {
		log.Fatalf("Failed to initialize cache: %s", err)
	}
	return cache
}
