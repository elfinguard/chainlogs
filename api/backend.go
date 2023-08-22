package api

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/holiman/uint256"
	"github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/chains"
)

var _ BackendService = &apiBackend{}

type apiBackend struct {
	vc         *chains.VirtualChain
	txFeed     event.Feed
	rmLogsFeed event.Feed
}

func NewBackend(vc *chains.VirtualChain) BackendService {
	return &apiBackend{
		vc: vc,
	}
}

func (backend *apiBackend) ChainId() *big.Int {
	return uint256.NewInt(0).SetBytes32(backend.vc.ChainID[:]).ToBig()
}

func (backend *apiBackend) BlockByNumber(number int64) (*types.Block, error) {
	s := backend.vc.Store
	//defer s.Close()
	block, err := s.GetBlockByHeight(uint64(number))
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (backend *apiBackend) BlockByHash(hash common.Hash) (*types.Block, error) {
	s := backend.vc.Store
	//defer s.Close()
	block, err := s.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (backend *apiBackend) GetTx(txHash common.Hash) (*types.Transaction, [65]byte, error) {
	return backend.vc.Store.GetTxByHash(txHash)
}

func (backend *apiBackend) GetTxListByHeight(height uint32) (txs []*types.Transaction, sigs [][65]byte, err error) {
	return backend.GetTxListByHeightWithRange(height, 0, math.MaxInt32)
}

func (backend *apiBackend) GetTxListByHeightWithRange(height uint32, start, end int) (txs []*types.Transaction, sigs [][65]byte, err error) {
	txs, sigs, err = backend.vc.Store.GetTxListByHeightWithRange(height, start, end)
	if err != nil {
		return
	}
	for _, tx := range txs {
		for i := 0; i < len(tx.Logs); i++ {
			l := &tx.Logs[i]
			if len(l.Data) >= 32 {
				hash := tx.Hash
				for i, j := 0, 31; i < j; i, j = i+1, j-1 {
					hash[i], hash[j] = hash[j], hash[i]
				}
				c := backend.vc.GetConfirmations(hash)
				latestConfirmations := uint256.NewInt(uint64(c)).Bytes32()
				if c == -1 {
					latestConfirmations = uint256.NewInt(0).SetAllOne().Bytes32()
				}
				copy(l.Data[:32], latestConfirmations[:])
			}
		}
	}
	return
}

func (backend *apiBackend) GetRpcMaxLogResults() int {
	return 2000
}

func (backend *apiBackend) QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter types.FilterFunc) ([]types.Log, error) {
	logs, err := backend.vc.Store.QueryLogs(addresses, topics, startHeight, endHeight, filter)
	if err != nil {
		return logs, err
	}
	for i := 0; i < len(logs); i++ {
		l := &logs[i]
		if len(l.Data) >= 32 {
			h := l.TxHash
			// flip txHash, as of bch txHash is reverse of txid
			for i, j := 0, 31; i < j; i, j = i+1, j-1 {
				h[i], h[j] = h[j], h[i]
			}
			c := backend.vc.GetConfirmations(h)
			latestConfirmations := uint256.NewInt(uint64(c)).Bytes32()
			if c == -1 {
				latestConfirmations = uint256.NewInt(0).SetAllOne().Bytes32()
			}
			copy(l.Data[:32], latestConfirmations[:])
		}
	}
	return logs, nil
}

func (backend *apiBackend) LatestHeight() int64 {
	height, _, _, _ := backend.vc.Store.GetLatestBlockInfo()
	return height
}

func (backend *apiBackend) SubscribeChainEvent(ch chan<- types.ChainEvent) event.Subscription {
	return backend.vc.SubscribeChainEvent(ch)
}
func (backend *apiBackend) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return backend.vc.SubscribeLogsEvent(ch)
}
func (backend *apiBackend) SubscribeNewTxsEvent(ch chan<- gethcore.NewTxsEvent) event.Subscription {
	return backend.txFeed.Subscribe(ch)
}
func (backend *apiBackend) SubscribeRemovedLogsEvent(ch chan<- gethcore.RemovedLogsEvent) event.Subscription {
	return backend.rmLogsFeed.Subscribe(ch)
}
