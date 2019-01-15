package watcher

import (
	"log"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/WeTrustPlatform/account-indexer/service"
)

// Watcher interface for Watch()
type Watcher interface {
	Watch()
}

// NodeStatusWatcher watch status of geth node
type NodeStatusWatcher struct {
	indexRepo repository.IndexRepo
	batchRepo repository.BatchRepo
}

// NewNodeStatusWatcher create NodeStatusWatcher
func NewNodeStatusWatcher(indexRepo repository.IndexRepo, batchRepo repository.BatchRepo) NodeStatusWatcher {
	return NodeStatusWatcher{indexRepo: indexRepo, batchRepo: batchRepo}
}

// Watch entry point of this struct
func (n NodeStatusWatcher) Watch() {
	// Watcher every 5 minute -> 5*60/15 ~ 20 blocks
	ticker := time.NewTicker(common.GetConfig().WatcherInterval)
	for t := range ticker.C {
		log.Println("Watcher: Watch geth node status at", t)
		n.watch()
	}
}

func (n NodeStatusWatcher) watch() {
	lastBlock, err := n.indexRepo.GetLastBlock()
	if err != nil {
		log.Println("Watcher warning: error=", err.Error())
		return
	}
	createdAt := common.UnmarshallIntToTime(lastBlock.CreatedAt)
	blockTime := common.UnmarshallIntToTime(lastBlock.Time)
	// 20 minutes -> 80 blocks
	// TODO: remove hard-coded
	createdAtDelay := time.Since(createdAt)
	blockDelay := createdAt.Sub(blockTime)
	if createdAtDelay > common.GetConfig().OOSThreshold || blockDelay > common.GetConfig().OOSThreshold {
		// TODO: unit test
		// TODO: update event database
		log.Printf("Geth node is out of date, createdAtDelay=%v, blockDelay=%v \n", createdAtDelay, blockDelay)
		service.GetIpcManager().ChangeIPC()
	}
}
