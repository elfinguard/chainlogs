package bch

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/rpcclient"
	"github.com/tendermint/tendermint/libs/log"
)

var _ IBchClient = &RetryableClient{}

type IBchClient interface {
	GetRawMempool() ([]*chainhash.Hash, error)
	GetRawTransactionVerbose(txHash *chainhash.Hash) (*btcjson.TxRawResult, error)
	GetTransaction(txHash *chainhash.Hash) (*btcjson.GetTransactionResult, error)
	GetBlockCount() (int64, error)
	GetBlockHash(blockHeight int64) (*chainhash.Hash, error)
	GetBlockVerboseTx(blockHash *chainhash.Hash) (*btcjson.GetBlockVerboseTxResult, error)
	TestMempoolAccept(rawTx []byte) (bool, error)
	SendRawTransaction(rawTx []byte) (*chainhash.Hash, error)
	GetTxOut(txHash *chainhash.Hash, index uint32, mempool bool) (*btcjson.GetTxOutResult, error)
}

type RetryableClient struct {
	connCfg  *rpcclient.ConnConfig
	client   *rpcclient.Client
	delay    int64 // sleep delay second when retry
	maxRetry int
	logger   log.Logger
}

func NewRetryableClient(mainChainClientInfo string, delayTime int64, maxRetry int, logger log.Logger) *RetryableClient {
	cfg, client := MakeMainChainClient(mainChainClientInfo)
	return &RetryableClient{
		connCfg:  cfg,
		client:   client,
		delay:    delayTime,
		maxRetry: maxRetry,
		logger:   logger,
	}
}

func MakeMainChainClient(mainChainClientInfo string) (*rpcclient.ConnConfig, *rpcclient.Client) {
	bchClientParams := strings.Split(mainChainClientInfo, ",")
	if len(bchClientParams) != 3 {
		panic("invalid main chain client param")
	}
	connCfg := &rpcclient.ConnConfig{
		Host:         bchClientParams[0],
		User:         bchClientParams[1],
		Pass:         bchClientParams[2],
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	mainChainClient, err := rpcclient.New(connCfg, nil)
	if err != nil {
		panic(err)
	}
	return connCfg, mainChainClient
}

func (r *RetryableClient) Delay() {
	time.Sleep(time.Duration(r.delay) * time.Second)
}

func (r *RetryableClient) GetRawMempool() (hashes []*chainhash.Hash, err error) {
	for i := 0; i < r.maxRetry; i++ {
		hashes, err = r.client.GetRawMempool()
		if err == nil {
			r.logger.Debug("getRawMempool", "error", err)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) GetRawTransactionVerbose(txHash *chainhash.Hash) (res *btcjson.TxRawResult, err error) {
	for i := 0; i < r.maxRetry; i++ {
		res, err = r.client.GetRawTransactionVerbose(txHash)
		if err == nil {
			r.logger.Debug("getRawTransactionVerbose", "txHash", txHash.String(), "error", err)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) GetTransaction(txHash *chainhash.Hash) (res *btcjson.GetTransactionResult, err error) {
	for i := 0; i < r.maxRetry; i++ {
		res, err = r.client.GetTransaction(txHash)
		if err == nil {
			r.logger.Debug("getTransaction", "txHash", txHash.String(), "error", err)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) GetBlockCount() (c int64, err error) {
	for i := 0; i < r.maxRetry; i++ {
		c, err = r.client.GetBlockCount()
		if err == nil {
			r.logger.Debug("getBlockCount", "error", err)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) GetBlockHash(blockHeight int64) (hash *chainhash.Hash, err error) {
	for i := 0; i < r.maxRetry; i++ {
		hash, err = r.client.GetBlockHash(blockHeight)
		if err == nil {
			r.logger.Debug("getBlockHash", "blockHeight", blockHeight, "error", err)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) GetBlockVerboseTx(blockHash *chainhash.Hash) (res *btcjson.GetBlockVerboseTxResult, err error) {
	for i := 0; i < r.maxRetry; i++ {
		res, err = r.client.GetBlockVerboseTx(blockHash)
		if err == nil {
			r.logger.Debug("getBlockVerboseTx", "blockHash", blockHash, "error", err)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) TestMempoolAccept(rawTx []byte) (ok bool, err error) {
	for i := 0; i < r.maxRetry; i++ {
		ok, err = testMempoolAccept("http://"+r.connCfg.Host, r.connCfg.User, r.connCfg.Pass, rawTx)
		if err == nil {
			r.logger.Debug("testMempoolAccept", "ok", ok)
			return ok, err
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) SendRawTransaction(rawTx []byte) (txHash *chainhash.Hash, err error) {
	for i := 0; i < r.maxRetry; i++ {
		txHash, err = r.client.SendRawSerializedTransaction(hex.EncodeToString(rawTx), true)
		if err == nil {
			r.logger.Debug("sendRawTransaction", "txHash", txHash)
			return
		}
		r.Delay()
	}
	return
}

func (r *RetryableClient) GetTxOut(txHash *chainhash.Hash, index uint32, mempool bool) (txOut *btcjson.GetTxOutResult, err error) {
	for i := 0; i < r.maxRetry; i++ {
		txOut, err = r.client.GetTxOut(txHash, index, mempool)
		if err == nil {
			r.logger.Debug("GetTxOut", "txHash", txHash, "index", index, "mempool", mempool)
			return
		}
		r.Delay()
	}
	return
}
