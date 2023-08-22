package chains

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	modbtypes "github.com/smartbch/moeingdb/types"
	evmtypes "github.com/smartbch/moeingevm/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/elfinguard/chainlogs/config"
	"github.com/elfinguard/chainlogs/scanner"
	"github.com/elfinguard/chainlogs/store"
)

type ChainLogs struct {
	Config *config.Config
	Chains map[string]*VirtualChain

	logger log.Logger
}

func NewChainLogs(config *config.Config, logger log.Logger) *ChainLogs {
	a := ChainLogs{
		Config: config,
		Chains: make(map[string]*VirtualChain),
		logger: logger,
	}
	return &a
}

func (a *ChainLogs) RegisterChain(chainName string, chain *VirtualChain) {
	a.Chains[chainName] = chain
}

func (a *ChainLogs) Run() {
	l := len(a.Chains)
	for name, c := range a.Chains {
		a.logger.Info("start chain", "name", name)
		if l == 1 {
			c.Start()
		} else {
			go c.Start()
		}
	}
}

type VirtualChain struct {
	Scanner                     scanner.IScanner
	Store                       store.IStore
	BlockInterval               int64
	ChainName                   string
	ChainID                     [32]byte
	GenesisMainChainBlockHeight int64

	PrevBlockHash         [32]byte
	CurrentBlockHeight    int64
	CurrentBlockTimestamp int64
	CurrentBlockHash      [32]byte

	ticker *time.Ticker

	chainFeed event.Feed // For pub&sub new blocks
	logsFeed  event.Feed // For pub&sub new logs
	scope     event.SubscriptionScope

	logger log.Logger
}

// for test
func (v *VirtualChain) SetLogger(logger log.Logger) {
	v.logger = logger
}

func (v *VirtualChain) Start() {
	v.RecoveryFromDB()
	now := time.Now().Unix()
	if now < v.CurrentBlockTimestamp {
		panic(fmt.Sprintf("now[%d] <= v.CurrentBlockTimestamp[%d]", now, v.CurrentBlockTimestamp))
	}
	downtime := now - v.CurrentBlockTimestamp
	if downtime >= v.BlockInterval {
		v.GenerateNewBlock(true)
	} else {
		time.Sleep(time.Duration(v.BlockInterval-downtime) * time.Second)
		v.GenerateNewBlock(true)
	}
	v.ticker = time.NewTicker(time.Duration(v.BlockInterval) * time.Second)
	tryGenerateBlockCount := 0
	for range v.ticker.C {
		tryGenerateBlockCount++
		v.GenerateNewBlock(tryGenerateBlockCount%12 == 0)
	}
}

func (v *VirtualChain) Stop() {
	v.scope.Close()
	v.ticker.Stop()
}

func (v *VirtualChain) RecoveryFromDB() {
	height, timestamp, hash, latestScanBlockHeight := v.Store.GetLatestBlockInfo()
	v.CurrentBlockHeight = height
	v.CurrentBlockTimestamp = timestamp
	v.CurrentBlockHash = hash
	v.PrevBlockHash = hash
	if latestScanBlockHeight == 0 {
		latestScanBlockHeight = v.GenesisMainChainBlockHeight
	}
	v.Scanner.SetLatestScanHeight(latestScanBlockHeight)
	v.logger.Info("recover virtual chain from db", "currentHeight", v.CurrentBlockHeight, "currentBlockTS", v.CurrentBlockTimestamp,
		"currentBlockHash", hex.EncodeToString(hash[:]), "latestScanBlockHeight", latestScanBlockHeight)
}

func (v *VirtualChain) GenerateNewBlock(scanBlock bool) {
	currentBlockTimestamp := time.Now().Unix()
	currentBlockHash := sha256.Sum256([]byte(v.ChainName + fmt.Sprintf(":%d", v.CurrentBlockHeight)))
	txs := v.Scanner.GetNewTxs(v.CurrentBlockHeight+1, currentBlockHash, scanBlock)
	if len(txs) == 0 && !scanBlock {
		v.logger.Debug("EGTX not found in this round")
		return
	}
	v.CurrentBlockHeight++
	v.CurrentBlockTimestamp = currentBlockTimestamp
	v.CurrentBlockHash = currentBlockHash

	evmBlk := evmtypes.Block{
		Number:     v.CurrentBlockHeight,
		Hash:       v.CurrentBlockHash,
		ParentHash: v.PrevBlockHash,
		// Different adaptors will get different timestamps. But we don't want authorizers disagree
		// on timestamp. So the timestamp is store in 'Size' which is ignored by authorizers
		Timestamp: 0,
		Size:      v.CurrentBlockTimestamp,
		GasUsed:   uint64(v.Scanner.GetLatestScanHeight()), // using gasUsed to store latest scanned mainnet block height
	}
	for _, tx := range txs {
		evmBlk.Transactions = append(evmBlk.Transactions, tx.HashId)
	}
	blkInfo, err := evmBlk.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	blk := modbtypes.Block{
		Height:    v.CurrentBlockHeight,
		BlockHash: v.CurrentBlockHash,
		BlockInfo: blkInfo,
		TxList:    txs,
	}
	v.Store.AddBlock(&blk)
	v.publishNewBlock(&blk)
	v.logger.Info("generate new block", "height", v.CurrentBlockHeight, "txs", len(txs), "blockHash", hex.EncodeToString(v.CurrentBlockHash[:]))
}

func (v *VirtualChain) GetConfirmations(txHash [32]byte) int32 {
	return v.Scanner.GetConfirmations(txHash)
}

func (v *VirtualChain) SubscribeChainEvent(ch chan<- evmtypes.ChainEvent) event.Subscription {
	return v.scope.Track(v.chainFeed.Subscribe(ch))
}

func (v *VirtualChain) SubscribeLogsEvent(ch chan<- []*gethtypes.Log) event.Subscription {
	return v.scope.Track(v.logsFeed.Subscribe(ch))
}

func (v *VirtualChain) publishNewBlock(mdbBlock *modbtypes.Block) {
	if mdbBlock == nil {
		return
	}
	chainEvent := evmtypes.BlockToChainEvent(mdbBlock)
	v.chainFeed.Send(chainEvent)
	if len(chainEvent.Logs) > 0 {
		v.logsFeed.Send(chainEvent.Logs)
	}
}

func TrapSignal(cleanupFunc func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		if cleanupFunc != nil {
			cleanupFunc()
		}
		exitCode := 128
		switch sig {
		case syscall.SIGINT:
			exitCode += int(syscall.SIGINT)
		case syscall.SIGTERM:
			exitCode += int(syscall.SIGTERM)
		}
		os.Exit(exitCode)
	}()
}
