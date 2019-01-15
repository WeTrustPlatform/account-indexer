package common

import (
	"fmt"
	"sync"
	"time"
)

// Configuration for the indexer
type configuration struct {
	CleanInterval   time.Duration
	BlockTTL        time.Duration
	WatcherInterval time.Duration
	// OOSThreshold threshold
	OOSThreshold time.Duration
	Port         int
	NumBatch     int
}

func (con *configuration) String() string {
	return fmt.Sprintf("CleanInterval=%v BlockTTL=%v WatcherInterval=%v OOSThreshold=%v Port=%v NumBatch=%v",
		con.CleanInterval, con.BlockTTL, con.WatcherInterval, con.OOSThreshold, con.Port, con.NumBatch)
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
