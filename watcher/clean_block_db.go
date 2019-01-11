package watcher

import (
	"log"
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/repository"
)

// Cleaner cleaner for account indexer
type Cleaner struct {
	repo repository.Repository
}

// NewCleaner create a cleaner instance
func NewCleaner(repo repository.Repository) Cleaner {
	return Cleaner{repo: repo}
}

// CleanBlockDB clean block db regularly
func (c Cleaner) CleanBlockDB() {
	// Clean every 5 minute -> 5*60/15 ~ 20 blocks
	ticker := time.NewTicker(5 * time.Minute)
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
	untilTime := lastUpdated.Add(-4 * time.Hour)
	total, err := c.repo.DeleteOldBlocks(big.NewInt(untilTime.Unix()))
	if err != nil {
		log.Println("Cleander: Deleting old blocks have error", err.Error())
	}
	log.Printf("Cleander: deleted %v blocks before %v \n", total, untilTime)
}
