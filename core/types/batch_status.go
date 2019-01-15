package types

import (
	"fmt"
	"math/big"
)

// BatchStatus the init batch status
type BatchStatus struct {
	// Block information for each batch
	// Key
	From      *big.Int
	To        *big.Int
	CreatedAt *big.Int
	Step      byte
	// Value
	Current   *big.Int
	UpdatedAt *big.Int
}

// IsDone the batch is done or not
func (bs *BatchStatus) IsDone() bool {
	if bs.Current == nil {
		return false
	}
	next := bs.Current.Int64() + int64(bs.Step)
	return next > bs.To.Int64()
}

// Next Return the next block number to run
func (bs *BatchStatus) Next() *big.Int {
	if bs.Current == nil {
		bs.Current = new(big.Int)
		bs.Current.Set(bs.From)
		return bs.Current
	}
	current := bs.Current.Int64()
	current = current + int64(bs.Step)
	bs.Current = bs.Current.SetInt64(current)
	return bs.Current
}

func (bs BatchStatus) String() string {
	return fmt.Sprintf("From %v, To %v, Step %v, Current %v, CreatedAt %v, UpdatedAt %v", bs.From, bs.To, bs.Step, bs.Current, bs.CreatedAt, bs.UpdatedAt)
}
