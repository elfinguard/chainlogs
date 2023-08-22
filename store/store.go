package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/moeingdb/modb"
	"github.com/smartbch/moeingdb/types"
	evmtypes "github.com/smartbch/moeingevm/types"
	"github.com/tendermint/tendermint/libs/log"
)

type IStore interface {
	AddBlock(blk *types.Block)
	//QueryLogs(addrOrList [][20]byte, topicsOrList [][][32]byte, startHeight, endHeight uint32, fn func([]byte) bool) error
	IsTxMined(txHash string) bool
	GetLatestBlockInfo() (height, timestamp int64, hash [32]byte, latestScanBlockHeight int64)
	GetBlockByHeight(height uint64) (*evmtypes.Block, error)
	GetBlockByHash(blkHash [32]byte) (blk *evmtypes.Block, err error)
	GetTxByHash(txHash common.Hash) (tx *evmtypes.Transaction, sig [65]byte, err error)
	GetTxListByHeightWithRange(height uint32, start, end int) (txs []*evmtypes.Transaction, sigs [][65]byte, err error)
	QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter evmtypes.FilterFunc) (logs []evmtypes.Log, err error)
	Close()
}

type ChainLogDB struct {
	modb *modb.MoDB
}

func NewChainLogDB(dataPath string, maxLogResults int, logger log.Logger) *ChainLogDB {
	var a ChainLogDB
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		_ = os.MkdirAll(path.Join(dataPath, "data"), 0700)
		var seed [8]byte // use current time as db's hash seed
		binary.LittleEndian.PutUint64(seed[:], uint64(time.Now().UnixNano()))
		a.modb = modb.CreateEmptyMoDB(dataPath, seed, logger)
	} else {
		a.modb = modb.NewMoDB(dataPath, logger)
	}
	a.modb.SetMaxEntryCount(maxLogResults)
	return &a
}

func (a *ChainLogDB) GetBlockByHeight(height uint64) (*evmtypes.Block, error) {
	bz := a.modb.GetBlockByHeight(int64(height))
	if len(bz) == 0 {
		return nil, evmtypes.ErrBlockNotFound
	}

	blk := &evmtypes.Block{}
	_, err := blk.UnmarshalMsg(bz)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (a *ChainLogDB) GetBlockByHash(blkHash [32]byte) (blk *evmtypes.Block, err error) {
	a.modb.GetBlockByHash(blkHash, func(bz []byte) bool {
		tmp := &evmtypes.Block{}
		_, err := tmp.UnmarshalMsg(bz)
		if err == nil && bytes.Equal(blkHash[:], tmp.Hash[:]) {
			blk = tmp
			return true // stop retry
		}
		return false
	})
	if blk == nil {
		err = evmtypes.ErrBlockNotFound
	}
	return
}

func (a *ChainLogDB) GetTxByHash(txHash common.Hash) (tx *evmtypes.Transaction, sig [65]byte, err error) {
	a.modb.GetTxByHash(txHash, func(b []byte) bool {
		tmp := &evmtypes.Transaction{}
		_, err := tmp.UnmarshalMsg(b[65:])
		if err == nil && bytes.Equal(tmp.Hash[:], txHash[:]) {
			tx = tmp
			copy(sig[:], b[:65])
			return true // stop retry
		}
		return false
	})
	if tx == nil {
		err = errors.New("tx not found")
	}
	return
}

func (a *ChainLogDB) GetTxListByHeightWithRange(height uint32, start, end int) (txs []*evmtypes.Transaction, sigs [][65]byte, err error) {
	txContents := a.modb.GetTxListByHeightWithRange(int64(height), start, end)
	txs = make([]*evmtypes.Transaction, len(txContents))
	sigs = make([][65]byte, len(txContents))
	for i, txContent := range txContents {
		copy(sigs[i][:], txContent[:65])
		txs[i] = &evmtypes.Transaction{}
		_, err = txs[i].UnmarshalMsg(txContent[65:])
		if err != nil {
			break
		}
	}
	return
}

func (a *ChainLogDB) QueryLogs(addresses []common.Address, topics [][]common.Hash, startHeight, endHeight uint32, filter evmtypes.FilterFunc) (logs []evmtypes.Log, err error) {
	rawAddresses := evmtypes.FromGethAddreses(addresses)
	rawTopics := make([][][32]byte, len(topics))
	for i, t := range topics {
		rawTopics[i] = evmtypes.FromGethHashes(t)
	}

	err = a.modb.QueryLogs(rawAddresses, rawTopics, startHeight, endHeight, func(data []byte) bool {
		if data == nil {
			err = errors.New("too many entities")
			return false
		}
		tx := evmtypes.Transaction{}
		if _, err = tx.UnmarshalMsg(data[65:]); err != nil {
			return false
		}

		var topicArr [4]common.Hash
		for _, l := range tx.Logs {
			for i, topic := range l.Topics {
				topicArr[i] = common.Hash(topic)
			}
			if filter(common.Address(l.Address), topicArr[:len(l.Topics)], addresses, topics) {
				logs = append(logs, l)
			}
		}
		return true
	})
	return
}

func (a *ChainLogDB) IsTxMined(txHash string) bool {
	_, _, err := a.GetTxByHash(common.HexToHash(txHash))
	isMined := err == nil
	return isMined
}

func (a *ChainLogDB) AddBlock(blk *types.Block) {
	a.modb.AddBlock(blk, -1, nil)
}

//func (a *ChainLogDB) QueryLogs(addrOrList [][20]byte, topicsOrList [][][32]byte, startHeight, endHeight uint32, fn func([]byte) bool) error {
//	return a.modb.QueryLogs(addrOrList, topicsOrList, startHeight, endHeight, fn)
//}

func (a *ChainLogDB) GetLatestBlockInfo() (height, timestamp int64, hash [32]byte, latestScanBlockHeight int64) {
	height = a.modb.GetLatestHeight()
	bz := a.modb.GetBlockByHeight(height)
	if len(bz) == 0 {
		return 0, time.Now().Unix(), hash, 0
	}
	blk := &evmtypes.Block{}
	_, err := blk.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	timestamp = blk.Timestamp
	hash = blk.Hash
	latestScanBlockHeight = int64(blk.GasUsed)
	return
}

func (a *ChainLogDB) Close() {
	a.modb.Close()
}

var _ IStore = &ChainLogDB{}
