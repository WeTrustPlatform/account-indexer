package common

import (
	"sync"
	"time"
)

// Configuration for the indexer
type configuration struct {
	CleanInterval time.Duration
	BlockTTL      time.Duration
}

// In minute
const DefaultCleanInterval = 5

// In hour
const DefaultBlockTTL = 4

var config *configuration
var once sync.Once

// GetConfig Singleton
func GetConfig() *configuration {
	once.Do(func() {
		config = &configuration{}
	})
	return config
}
