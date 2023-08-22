package store

import (
	"os"
	"testing"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/testutils"
)

const (
	dbPath = "./testdb"
)

func TestGetTx(t *testing.T) {
	db := NewChainLogDB(dbPath, 100, log.NewNopLogger())
	defer os.RemoveAll(dbPath)
	defer db.Close()

	addr1 := gethcmn.Address{0xA1, 0x23}
	addr2 := gethcmn.Address{0xA2, 0x34}
	topic1 := gethcmn.Hash{0xD1, 0x23}
	topic2 := gethcmn.Hash{0xD2, 0x34}
	topic3 := gethcmn.Hash{0xD3, 0x45}
	topic4 := gethcmn.Hash{0xD4, 0x56}

	blk := testutils.NewMdbBlockBuilder().
		Height(1).Hash(gethcmn.Hash{0xB1}).
		Tx(gethcmn.Hash{0xC1}, types.Log{
			Address: addr1,
			Topics:  [][32]byte{topic1, topic2},
		}).
		Tx(gethcmn.Hash{0xC2}, types.Log{
			Address: addr2,
			Topics:  [][32]byte{topic3, topic4},
		}).
		Build()
	db.AddBlock(blk)
	db.AddBlock(nil)

	blk1, err := db.GetBlockByHash(gethcmn.Hash{0xB1})
	require.NoError(t, err)
	require.Len(t, blk1.Transactions, 2)

	tx1, _, err := db.GetTxByHash(gethcmn.Hash{0xC1})
	require.NoError(t, err)
	require.Len(t, tx1.Logs, 1)
}
