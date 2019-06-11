package watcher

import (
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/common/config"
	"github.com/WeTrustPlatform/account-indexer/repository"
	log "github.com/sirupsen/logrus"
)

// Cleaner cleaner for account indexer
type Cleaner struct {
	repo repository.IndexRepo
}

// NewCleaner create a cleaner instance
func NewCleaner(repo repository.IndexRepo) Cleaner {
	return Cleaner{repo: repo}
}

// CleanBlockDB clean block db regularly
func (c Cleaner) CleanBlockDB() {
	// Clean every 5 minute -> 5*60/15 ~ 20 blocks
	ticker := time.NewTicker(config.GetConfig().CleanInterval)
	for t := range ticker.C {
		log.WithField("ticker", t).Info("Cleaner: Clean Block DB")
		c.cleanBlockDB()
	}
}

func (c Cleaner) cleanBlockDB() {
	lastBlock, err := c.repo.GetLastBlock()
	if err != nil {
		log.WithField("error", err.Error()).Error("Cleaner error")
		return
	}
	lastUpdated := common.UnmarshallIntToTime(lastBlock.CreatedAt)
	untilTime := lastUpdated.Add(-config.GetConfig().BlockTTL)
	total, err := c.repo.DeleteOldBlocks(big.NewInt(untilTime.Unix()))
	if err != nil {
		log.WithField("error", err.Error()).Error("Cleaner: Deleting old blocks have error")
	}
	log.WithFields(log.Fields{
		"total": total,
		"until": untilTime,
	}).Info("Cleaner: deleted blocks")
}
