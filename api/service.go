package api

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/smartbch/moeingevm/types"
	motypes "github.com/smartbch/moeingevm/types"
	"math/big"
)

type CallDetail struct {
	Status                 int
	GasUsed                uint64
	OutData                []byte
	Logs                   []motypes.EvmLog
	CreatedContractAddress common.Address
	InternalTxCalls        []motypes.InternalTxCall
	InternalTxReturns      []motypes.InternalTxReturn
	RwLists                *motypes.ReadWriteLists
}

type FilterService interface {
	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
	SubscribeChainEvent(ch chan<- motypes.ChainEvent) event.Subscription
	SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription
	SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription
}

type BackendService interface {
	FilterService
	GetTx(txHash common.Hash) (*types.Transaction, [65]byte, error)
	GetTxListByHeight(height uint32) (tx []*motypes.Transaction, sigs [][65]byte, err error)
	GetRpcMaxLogResults() int
	BlockByNumber(number int64) (*motypes.Block, error)
	BlockByHash(hash common.Hash) (*types.Block, error)
	LatestHeight() int64
	QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter types.FilterFunc) ([]types.Log, error)
	ChainId() *big.Int
}
