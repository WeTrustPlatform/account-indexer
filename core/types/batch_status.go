package types

import (
	"fmt"
	"math/big"
)

// BatchStatus the init batch status
type BatchStatus struct {
	// Block information for each batch
	From      *big.Int
	To        *big.Int
	Current   *big.Int
	CreatedAt *big.Int
	UpdatedAt *big.Int
}

// IsDone the batch is done or not
func (bs BatchStatus) IsDone() bool {
	return bs.Current != nil && bs.To.Cmp(bs.Current) <= 0
}

func (bs BatchStatus) String() string {
	return fmt.Sprintf("From %v, To %v, Current %v, CreatedAt %v, UpdatedAt %v", bs.From, bs.To, bs.Current, bs.CreatedAt, bs.UpdatedAt)
}
