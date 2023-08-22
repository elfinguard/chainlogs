package api

import (
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/api"
)

const (
	BlockMaxGas int64 = 1_000_000_000 // 1Billion
)

type PublicEthAPI interface {
	ChainId() hexutil.Uint64
	BlockNumber() (hexutil.Uint64, error)
	GetBlockByNumber(blockNum gethrpc.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetTransactionByHash(hash gethcmn.Hash) (*Transaction, error)
}

type ethAPI struct {
	backend api.BackendService
	logger  log.Logger
}

func newEthAPI(backend api.BackendService, logger log.Logger) *ethAPI {
	return &ethAPI{
		backend: backend,
		logger:  logger,
	}
}

func (api *ethAPI) ChainId() hexutil.Uint64 {
	return hexutil.Uint64(api.backend.ChainId().Uint64())
}

// https://eth.wiki/json-rpc/API#eth_blockNumber
func (api *ethAPI) BlockNumber() (hexutil.Uint64, error) {
	return hexutil.Uint64(api.backend.LatestHeight()), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockByNumber
func (api *ethAPI) GetBlockByNumber(blockNum gethrpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.backend.BlockByNumber(int64(blockNum))
	if err != nil {
		if err == mevmtypes.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}

	//var txs []*types.Transaction
	//var sigs [][65]byte
	//if fullTx {
	//	txs, sigs, err = api.backend.GetTxListByHeight(uint32(block.Number))
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	return blockToRpcResp(block), nil
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByHash
func (api *ethAPI) GetTransactionByHash(hash gethcmn.Hash) (*Transaction, error) {
	tx, _, err := api.backend.GetTx(hash)
	if err != nil {
		return nil, nil
	}
	return txToRpcResp(tx), nil
}

func blockToRpcResp(block *mevmtypes.Block) map[string]interface{} {
	result := map[string]interface{}{
		"number":           hexutil.Uint64(block.Number),
		"hash":             hexutil.Bytes(block.Hash[:]),
		"parentHash":       hexutil.Bytes(block.ParentHash[:]),
		"nonce":            hexutil.Bytes(make([]byte, 8)), // PoW specific
		"sha3Uncles":       gethcmn.Hash{},                 // No uncles in Tendermint
		"logsBloom":        gethtypes.Bloom{},
		"transactionsRoot": hexutil.Bytes(block.TransactionsRoot[:]),
		"stateRoot":        hexutil.Bytes(block.StateRoot[:]),
		"miner":            hexutil.Bytes(block.Miner[:]),
		"mixHash":          gethcmn.Hash{},
		"difficulty":       hexutil.Uint64(0),
		"totalDifficulty":  hexutil.Uint64(0),
		"extraData":        hexutil.Bytes(nil),
		"size":             hexutil.Uint64(block.Size),
		"gasLimit":         hexutil.Uint64(BlockMaxGas),
		"gasUsed":          hexutil.Uint64(block.GasUsed),
		"timestamp":        hexutil.Uint64(block.Timestamp),
		"transactions":     mevmtypes.ToGethHashes(block.Transactions),
		"uncles":           []string{},
		"receiptsRoot":     gethcmn.Hash{},
	}

	//if len(txs) > 0 {
	//	rpcTxs := make([]*Transaction, len(txs))
	//	for i, tx := range txs {
	//		rpcTxs[i] = txToRpcResp(tx, sigs[i])
	//	}
	//	result["transactions"] = rpcTxs
	//}

	return result
}

func txToRpcResp(tx *mevmtypes.Transaction) *Transaction {
	idx := hexutil.Uint64(tx.TransactionIndex)
	resp := &Transaction{
		BlockHash:        &gethcmn.Hash{},
		BlockNumber:      (*hexutil.Big)(big.NewInt(tx.BlockNumber)),
		From:             tx.From,
		Gas:              hexutil.Uint64(tx.Gas),
		GasPrice:         (*hexutil.Big)(big.NewInt(0).SetBytes(tx.GasPrice[:])),
		Hash:             tx.Hash,
		Input:            tx.Input,
		Nonce:            hexutil.Uint64(tx.Nonce),
		TransactionIndex: &idx,
		Value:            (*hexutil.Big)(big.NewInt(0).SetBytes(tx.Value[:])),
		//V:                (*hexutil.Big)(v),
		//R:                (*hexutil.Big)(r),
		//S:                (*hexutil.Big)(s),
	}
	copy(resp.BlockHash[:], tx.BlockHash[:])
	if tx.To != [20]byte{} {
		resp.To = &gethcmn.Address{}
		copy(resp.To[:], tx.To[:])
	}
	return resp
}
