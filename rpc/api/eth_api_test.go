package api

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/elfinguard/chainlogs/testchain"
	"github.com/tendermint/tendermint/libs/log"
)

func TestBlockNum(t *testing.T) {
	vc := testchain.CreateTestChain()
	defer vc.Destroy()
	_api := newEthAPI(vc.NewBackend(), log.NewNopLogger())

	n, err := _api.BlockNumber()
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(0), n)

	vc.GenNewBlock()
	n, err = _api.BlockNumber()
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(1), n)

	x := 1 + rand.Intn(100)
	for i := 0; i < x; i++ {
		vc.GenNewBlock()
	}
	n, err = _api.BlockNumber()
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(1+x), n)
}

func TestGetBlockByNum_notFound(t *testing.T) {
	vc := testchain.CreateTestChain()
	defer vc.Destroy()
	_api := newEthAPI(vc.NewBackend(), log.NewNopLogger())

	vc.GenNewBlock()
	result, err := _api.GetBlockByNumber(100, false)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGetBlockByNum(t *testing.T) {
	vc := testchain.CreateTestChain()
	defer vc.Destroy()
	_api := newEthAPI(vc.NewBackend(), log.NewNopLogger())

	vc.AddTx(gethcmn.Hash{0xC1})
	vc.GenNewBlock()
	result, err := _api.GetBlockByNumber(1, false)
	require.NoError(t, err)
	delete(result, "size")
	require.Equal(t, `{
  "difficulty": "0x0",
  "extraData": "0x",
  "gasLimit": "0x3b9aca00",
  "gasUsed": "0x0",
  "hash": "0x457965f7c8e31dbcfde9c7e2a0612df8b69b4e2d88afda065712d501b4f892dc",
  "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "miner": "0x0000000000000000000000000000000000000000",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "nonce": "0x0000000000000000",
  "number": "0x1",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "receiptsRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "sha3Uncles": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "stateRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": "0x0",
  "totalDifficulty": "0x0",
  "transactions": [
    "0xc100000000000000000000000000000000000000000000000000000000000000"
  ],
  "transactionsRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "uncles": []
}`, toJSON(result))
}

func TestGetTxByHash(t *testing.T) {
	vc := testchain.CreateTestChain()
	defer vc.Destroy()
	_api := newEthAPI(vc.NewBackend(), log.NewNopLogger())

	txHash := gethcmn.Hash{0xC1}
	vc.AddTx(txHash)
	bNum, bHash := vc.GenNewBlock()

	tx, err := _api.GetTransactionByHash(txHash)
	require.NoError(t, err)
	require.Equal(t, tx.BlockHash, &bHash)
	require.Equal(t, tx.BlockNumber.ToInt().Int64(), bNum)
}

func toJSON(v interface{}) string {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
