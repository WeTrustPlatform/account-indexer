package watcher

import (
	"log"
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/common/config"
	"github.com/WeTrustPlatform/account-indexer/repository"
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
		log.Println("Cleaner: Clean Block DB at", t)
		c.cleanBlockDB()
	}
}

func (c Cleaner) cleanBlockDB() {
	lastBlock, err := c.repo.GetLastBlock()
	if err != nil {
		log.Println("Cleaner warning: error=", err.Error())
		return
	}
	lastUpdated := common.UnmarshallIntToTime(lastBlock.CreatedAt)
	untilTime := lastUpdated.Add(-config.GetConfig().BlockTTL)
	total, err := c.repo.DeleteOldBlocks(big.NewInt(untilTime.Unix()))
	if err != nil {
		log.Println("Cleaner: Deleting old blocks have error", err.Error())
	}
	log.Printf("Cleaner: deleted %v blocks before %v \n", total, untilTime)
}
