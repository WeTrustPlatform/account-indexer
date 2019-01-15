package watcher

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/core/types"
)

func TestShouldChangeIPC(t *testing.T) {
	config := common.GetConfig()
	config.OOSThreshold = 5 * time.Minute
	// both are good
	lastBlock := types.BlockIndex{
		CreatedAt: big.NewInt(time.Now().Add(-2 * time.Minute).Unix()),
		Time:      big.NewInt(time.Now().Add(-3 * time.Minute).Unix()),
	}
	change := shouldChangeIPC(lastBlock)
	assert.False(t, change)
	// both are good
	lastBlock = types.BlockIndex{
		CreatedAt: big.NewInt(time.Now().Add(-4 * time.Minute).Unix()),
		Time:      big.NewInt(time.Now().Add(-6 * time.Minute).Unix()),
	}
	change = shouldChangeIPC(lastBlock)
	assert.False(t, change)
	// block comes too late
	lastBlock = types.BlockIndex{
		CreatedAt: big.NewInt(time.Now().Add(-6 * time.Minute).Unix()),
		Time:      big.NewInt(time.Now().Add(-7 * time.Minute).Unix()),
	}
	change = shouldChangeIPC(lastBlock)
	assert.True(t, change)
	// block comes on time but it's not up-to-date
	lastBlock = types.BlockIndex{
		CreatedAt: big.NewInt(time.Now().Add(-4 * time.Minute).Unix()),
		Time:      big.NewInt(time.Now().Add(-10 * time.Minute).Unix()),
	}
	change = shouldChangeIPC(lastBlock)
	assert.True(t, change)
}
