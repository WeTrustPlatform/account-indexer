package watcher

import (
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/common/config"
	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/WeTrustPlatform/account-indexer/service"
	log "github.com/sirupsen/logrus"
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
		log.Info("Watcher: waching, no need to watch again")
		return
	}
	n.isWatching = true
	ticker := time.NewTicker(config.GetConfig().WatcherInterval)
	for t := range ticker.C {
		log.WithField("ticket", t).Info("Watcher: Watch geth node status")
		n.watch(ticker)
	}
}

func (n *NodeStatusWatcher) watch(ticker *time.Ticker) {
	lastBlock, err := n.indexRepo.GetLastBlock()
	if err != nil {
		log.WithField("error", err.Error).Error("Watcher.watch error")
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
	if createdAtDelay > config.GetConfig().OOSThreshold || blockDelay > config.GetConfig().OOSThreshold {
		log.WithFields(log.Fields{
			"createdAtDelay": createdAtDelay,
			"blockDelay":     blockDelay,
			"OOSThreshold":   config.GetConfig().OOSThreshold,
		}).Info("Watcher: Geth node is out of date")
		return true
	}
	return false
}
