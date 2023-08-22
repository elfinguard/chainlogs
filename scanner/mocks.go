package scanner

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	modbtypes "github.com/smartbch/moeingdb/types"
	"github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/store"
)

var _ store.IStore = &MockStore{}

type MockStore struct {
	txByHash      map[string]*types.Transaction
	blkByHash     map[[32]byte]*modbtypes.Block
	blkByHeight   map[int64]*modbtypes.Block
	latestHeight  int64
	timestamp     int64
	latestBlkHash [32]byte
}

func (m *MockStore) GetBlockByHeight(height uint64) (*types.Block, error) {
	panic("implement me")
}

func (m *MockStore) GetTxByHash(txHash common.Hash) (tx *types.Transaction, sig [65]byte, err error) {
	return
}

func (m *MockStore) AddBlock(blk *modbtypes.Block) {
	m.blkByHash[blk.BlockHash] = blk
	m.blkByHeight[blk.Height] = blk
	m.latestBlkHash = blk.BlockHash
	m.latestHeight = blk.Height
	m.timestamp = time.Now().Unix()
}

func (m *MockStore) IsTxMined(txHash string) bool {
	return m.txByHash[txHash] != nil
}

func (m *MockStore) GetLatestBlockInfo() (height, timestamp int64, hash [32]byte, latestScanBlockHeight int64) {
	return
}

func (m *MockStore) GetBlockByHash(blkHash [32]byte) (blk *types.Block, err error) {
	return
}

func (m MockStore) GetTxListByHeightWithRange(height uint32, start, end int) (txs []*types.Transaction, sigs [][65]byte, err error) {
	return
}

func (m MockStore) QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter types.FilterFunc) (logs []types.Log, err error) {
	return
}

func (m MockStore) Close() {
	//return
}
