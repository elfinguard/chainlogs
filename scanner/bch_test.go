package scanner

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchutil"
	"github.com/tendermint/tendermint/libs/log"

	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/bch"
)

func TestBchScanner(t *testing.T) {
	mc := &bch.MockClient{
		//txByHash: make(map[chainhash.Hash]*btcjson.TxRawResult),
	}
	b := BchScanner{
		Client: mc,
		Store: &MockStore{
			txByHash:      make(map[string]*types.Transaction),
			blkByHash:     make(map[[32]byte]*modbtypes.Block),
			blkByHeight:   make(map[int64]*modbtypes.Block),
			latestHeight:  0,
			timestamp:     0,
			latestBlkHash: [32]byte{},
		},
		MaxTxsInBlock:    1,
		OriginChainParam: &chaincfg.MainNetParams,
		logger:           log.NewNopLogger(),
	}
	tx, contractAddress, payer, payee, fileID, data := buildEGTx(mc)
	blkHash := [32]byte{0x1}
	mTx, err := b.convertUtxoInfoToTx(tx, 0, 1, blkHash)
	require.Nil(t, err)
	require.Equal(t, tx.Txid, hex.EncodeToString(mTx.HashId[:]))
	require.Equal(t, payer, mTx.SrcAddr)
	require.Equal(t, payee, mTx.DstAddr)
	var originTx types.Transaction
	_, err = originTx.UnmarshalMsg(mTx.Content[:])
	require.NoError(t, err)
	require.Equal(t, contractAddress, originTx.ContractAddress)
	require.Equal(t, 1, len(mTx.LogList))
	l := mTx.LogList[0]
	require.Equal(t, contractAddress, l.Address)
	require.True(t, bytes.Equal(payer[:], l.Topics[0][12:]))
	require.True(t, bytes.Equal(payee[:], l.Topics[1][12:]))
	require.Equal(t, fileID, l.Topics[2])

	require.Equal(t, 1, len(originTx.Logs))
	lg := originTx.Logs[0]
	require.Equal(t, contractAddress, lg.Address)
	require.True(t, bytes.Equal(payer[:], lg.Topics[0][12:]))
	require.True(t, bytes.Equal(payee[:], lg.Topics[1][12:]))
	require.Equal(t, fileID, lg.Topics[2])
	require.True(t, bytes.Equal(data[:], lg.Data[len(lg.Data)-32:]))
}

func buildEGTx(m *bch.MockClient) (*btcjson.TxRawResult, [20]byte, [20]byte, [20]byte, [32]byte, [32]byte) {
	payer := [20]byte{0x02}
	tx0Hash := [32]byte{0x01}
	tx0 := btcjson.TxRawResult{
		Txid: hex.EncodeToString(tx0Hash[:]),
		Hash: hex.EncodeToString(tx0Hash[:]),
	}
	vout := btcjson.Vout{
		Value: 1,
		ScriptPubKey: btcjson.ScriptPubKeyResult{
			Type: "pubkeyhash",
		},
	}
	addressPubkeyHash, _ := bchutil.NewAddressPubKeyHash(payer[:], &chaincfg.MainNetParams)
	vout.ScriptPubKey.Addresses = append(vout.ScriptPubKey.Addresses, addressPubkeyHash.EncodeAddress())
	script, _ := txscript.PayToAddrScript(addressPubkeyHash)
	vout.ScriptPubKey.Hex = hex.EncodeToString(script)
	tx0.Vout = append(tx0.Vout, vout)
	tx0H, _ := chainhash.NewHash(tx0Hash[:])
	m.AddTx(tx0H, &tx0)

	tx1Hash := [32]byte{0x02}
	tx1 := btcjson.TxRawResult{
		Txid: hex.EncodeToString(tx1Hash[:]),
		Hash: hex.EncodeToString(tx1Hash[:]),
	}
	vin := btcjson.Vin{Txid: tx0H.String()}
	tx1.Vin = append(tx1.Vin, vin)
	vout = btcjson.Vout{
		ScriptPubKey: btcjson.ScriptPubKeyResult{
			Asm:  "OP_RETURN EGTX",
			Type: "nulldata",
		},
	}
	var contractAddress = [20]byte{0x01}
	var payee = [20]byte{0x03}
	var fileID = [32]byte{0x01}
	var data = [32]byte{0x01}
	script, _ = txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte("EGTX")).
		AddData(contractAddress[:]).
		AddData(payer[:]).
		AddData(payee[:]).
		AddData(fileID[:]).
		AddOp(txscript.OP_FALSE).
		AddData(data[:]).Script()
	vout.ScriptPubKey.Hex = hex.EncodeToString(script)
	tx1.Vout = append(tx1.Vout, vout)
	vout = btcjson.Vout{
		Value: 1,
		ScriptPubKey: btcjson.ScriptPubKeyResult{
			Type: "pubkeyhash",
		},
	}
	addressPubkeyHash, _ = bchutil.NewAddressPubKeyHash(payee[:], &chaincfg.MainNetParams)
	vout.ScriptPubKey.Addresses = append(vout.ScriptPubKey.Addresses, addressPubkeyHash.EncodeAddress())
	script, _ = txscript.PayToAddrScript(addressPubkeyHash)
	vout.ScriptPubKey.Hex = hex.EncodeToString(script)
	tx1.Vout = append(tx1.Vout, vout)
	tx1H, _ := chainhash.NewHash(tx1Hash[:])
	m.AddTx(tx1H, &tx1)
	return &tx1, contractAddress, payer, payee, fileID, data
}
