package watcher

import (
	"log"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/WeTrustPlatform/account-indexer/service"
)

// Watcher interface for Watch()
type Watcher interface {
	Watch()
}

// NodeStatusWatcher watch status of geth node
type NodeStatusWatcher struct {
	indexRepo  repository.IndexRepo
	batchRepo  repository.BatchRepo
	isWatching bool
}

// NewNodeStatusWatcher create NodeStatusWatcher
func NewNodeStatusWatcher(indexRepo repository.IndexRepo, batchRepo repository.BatchRepo) NodeStatusWatcher {
	return NodeStatusWatcher{indexRepo: indexRepo, batchRepo: batchRepo}
}

// Watch entry point of this struct
func (n *NodeStatusWatcher) Watch() {
	if n.isWatching {
		// Don't start new ticker again
		log.Println("Watcher: waching, no need to watch again")
		return
	}
	n.isWatching = true
	ticker := time.NewTicker(common.GetConfig().WatcherInterval)
	for t := range ticker.C {
		log.Println("Watcher: Watch geth node status at", t)
		n.watch(ticker)
	}
}

func (n *NodeStatusWatcher) watch(ticker *time.Ticker) {
	lastBlock, err := n.indexRepo.GetLastBlock()
	if err != nil {
		log.Println("Watcher: warning: error=", err.Error())
		return
	}
	if shouldChangeIPC(lastBlock) {
		n.isWatching = false
		ticker.Stop()
		// TODO: update event database
		service.GetIpcManager().ChangeIPC()
	}
}

func shouldChangeIPC(lastBlock types.BlockIndex) bool {
	createdAt := common.UnmarshallIntToTime(lastBlock.CreatedAt)
	blockTime := common.UnmarshallIntToTime(lastBlock.Time)
	// check if no new block is received for a long time
	createdAtDelay := time.Since(createdAt)
	// check if the last received block is on time or not
	blockDelay := createdAt.Sub(blockTime)
	if createdAtDelay > common.GetConfig().OOSThreshold || blockDelay > common.GetConfig().OOSThreshold {
		log.Printf("Watcher: Geth node is out of date, createdAtDelay=%v, blockDelay=%v OOSThreshold=%v \n", createdAtDelay, blockDelay, common.GetConfig().OOSThreshold)
		return true
	}
	return false
}
