package filters

import (
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethfilters "github.com/ethereum/go-ethereum/eth/filters"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/testchain"
	"github.com/elfinguard/chainlogs/testutils"
)

func TestBackend(t *testing.T) {
	vc := testchain.CreateTestChain()
	defer vc.Destroy()

	addr1 := gethcmn.Address{0xA1, 0x23}
	addr2 := gethcmn.Address{0xA2, 0x34}
	topic1 := gethcmn.Hash{0xD1, 0x23}
	topic2 := gethcmn.Hash{0xD2, 0x34}
	topic3 := gethcmn.Hash{0xD3, 0x45}
	topic4 := gethcmn.Hash{0xD4, 0x56}

	vc.AddTx(gethcmn.Hash{0xC1}, mevmtypes.Log{
		Address: addr1,
		Topics:  [][32]byte{topic1, topic2},
	})
	vc.AddTx(gethcmn.Hash{0xC2}, mevmtypes.Log{
		Address: addr2,
		Topics:  [][32]byte{topic3, topic4},
	})

	_, b1Hash := vc.GenNewBlock()
	vc.WaitMS(50)

	backend := vc.NewBackend()
	b1, err := backend.BlockByHash(b1Hash)
	require.NoError(t, err)
	require.Len(t, b1.Transactions, 2)

	tx, _, err := backend.GetTx(b1.Transactions[0])
	require.NoError(t, err)
	require.Len(t, tx.Logs, 1)
}

func TestGetLogs_blockHashFilter(t *testing.T) {
	vc := testchain.CreateTestChain()
	defer vc.Destroy()

	_api := NewAPI(vc.NewBackend(), log.NewNopLogger())

	addr1 := gethcmn.Address{0xA1}
	addr2 := gethcmn.Address{0xA2}
	addr3 := gethcmn.Address{0xA3}
	addr4 := gethcmn.Address{0xA4}
	txId1 := gethcmn.Hash{0xC1}
	txId2 := gethcmn.Hash{0xC2}
	txId3 := gethcmn.Hash{0xC3}
	txId4 := gethcmn.Hash{0xC4}

	vc.AddTx(txId1, mevmtypes.Log{Address: addr1})
	vc.AddTx(txId2, mevmtypes.Log{Address: addr2})
	_, b1Hash := vc.GenNewBlock()

	vc.AddTx(txId3, mevmtypes.Log{Address: addr3})
	vc.AddTx(txId4, mevmtypes.Log{Address: addr4})
	_, b2Hash := vc.GenNewBlock()

	vc.WaitMS(50)
	logs, err := _api.GetLogs(testutils.NewBlockHashFilter(&b1Hash, addr1))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, addr1, logs[0].Address)

	logs, err = _api.GetLogs(testutils.NewBlockHashFilter(&b2Hash, addr3))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, addr3, logs[0].Address)

	logs, err = _api.GetLogs(testutils.NewBlockHashFilter(&b2Hash))
	require.NoError(t, err)
	require.Len(t, logs, 2)

	b3Hash := gethcmn.Hash{0xB3}
	_, err = _api.GetLogs(gethfilters.FilterCriteria{BlockHash: &b3Hash})
	require.Error(t, err)
}
