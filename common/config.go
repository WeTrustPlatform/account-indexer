package common

import (
	"sync"
	"time"
)

// Configuration for the indexer
type configuration struct {
	CleanInterval time.Duration
	BlockTTL      time.Duration
	Port          int
	NumBatch      int
}

var config *configuration
var once sync.Once

// GetConfig Singleton
func GetConfig() *configuration {
	once.Do(func() {
		config = &configuration{}
	})
	return config
}
