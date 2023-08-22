package bch

import (
	"encoding/hex"
	"fmt"

	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg/chainhash"
)

var _ IBchClient = &MockClient{}

type MockClient struct {
	txs        []*chainhash.Hash
	txByHash   map[chainhash.Hash]*btcjson.TxRawResult
	txToAccept map[string]bool
	txToSend   map[string]*chainhash.Hash
}

func (m *MockClient) AddTx(txHash *chainhash.Hash, tx *btcjson.TxRawResult) {
	m.txs = append(m.txs, txHash)
	if m.txByHash == nil {
		m.txByHash = map[chainhash.Hash]*btcjson.TxRawResult{}
	}
	m.txByHash[*txHash] = tx
}

func (m *MockClient) AddTxToAccept(rawTx string) {
	if m.txToAccept == nil {
		m.txToAccept = map[string]bool{}
	}
	m.txToAccept[rawTx] = true
}

func (m *MockClient) AddTxToSend(rawTx string, txHash *chainhash.Hash) {
	if m.txToSend == nil {
		m.txToSend = map[string]*chainhash.Hash{}
	}
	m.txToSend[rawTx] = txHash
}

func (m *MockClient) GetBlockVerboseTx(blockHash *chainhash.Hash) (*btcjson.GetBlockVerboseTxResult, error) {
	return nil, nil
}

func (m *MockClient) GetBlockCount() (int64, error) {
	return 0, nil
}

func (m *MockClient) GetBlockHash(blockHeight int64) (*chainhash.Hash, error) {
	return nil, nil
}

func (m *MockClient) GetRawMempool() ([]*chainhash.Hash, error) {
	return m.txs, nil
}

func (m *MockClient) GetRawTransactionVerbose(txHash *chainhash.Hash) (*btcjson.TxRawResult, error) {
	res := m.txByHash[*txHash]
	if res == nil {
		return nil, fmt.Errorf("tx not found: %s", txHash.String())
	}
	return res, nil
}

func (m *MockClient) GetTransaction(txHash *chainhash.Hash) (*btcjson.GetTransactionResult, error) {
	return nil, nil
}

func (m *MockClient) TestMempoolAccept(rawTx []byte) (bool, error) {
	return m.txToAccept[hex.EncodeToString(rawTx)], nil
}

func (m *MockClient) SendRawTransaction(rawTx []byte) (*chainhash.Hash, error) {
	return m.txToSend[hex.EncodeToString(rawTx)], nil
}
