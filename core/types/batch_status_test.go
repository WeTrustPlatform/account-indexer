package types

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNext(t *testing.T) {
	batch := &BatchStatus{
		From: big.NewInt(0),
		To:   big.NewInt(100),
		Step: 10,
	}
	current := batch.Next()
	assert.Equal(t, big.NewInt(0), current)
	current = batch.Next()
	assert.Equal(t, big.NewInt(10), current)
	current = batch.Next()
	assert.Equal(t, big.NewInt(20), current)
}

func TestIsDone(t *testing.T) {
	batch := &BatchStatus{
		From:    big.NewInt(0),
		To:      big.NewInt(10),
		Step:    3,
		Current: big.NewInt(9),
	}
	isDone := batch.IsDone()
	assert.True(t, isDone)
}
