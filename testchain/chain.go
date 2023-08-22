package testchain

import (
	"os"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/log"

	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/api"
	"github.com/elfinguard/chainlogs/chains"
	"github.com/elfinguard/chainlogs/config"
	"github.com/elfinguard/chainlogs/store"
)

const (
	dbPath = "./testdb"
)

type TestChain struct {
	*chains.VirtualChain
	scanner *FakeScanner
}

func CreateTestChain() *TestChain {
	_ = os.RemoveAll(dbPath)

	cfg := config.DefaultConfig()
	bchChainConfig := config.NewBchChainConfig(&cfg, []string{}, 0)
	cfg.RegisterChainConfig(bchChainConfig.ChainName, bchChainConfig)
	defaultRpcEthGetLogsMaxResults := 10000
	s := store.NewChainLogDB(dbPath, defaultRpcEthGetLogsMaxResults, log.NewNopLogger())
	a := chains.NewChainLogs(&cfg, log.NewNopLogger())
	fakeScanner := &FakeScanner{}
	bchVirtualChain := &chains.VirtualChain{
		Scanner:       fakeScanner,
		Store:         s,
		BlockInterval: 5,
		ChainName:     bchChainConfig.ChainName,
		ChainID:       bchChainConfig.ChainId,
	}
	bchVirtualChain.SetLogger(log.NewNopLogger())
	a.RegisterChain(bchChainConfig.ChainName, bchVirtualChain)

	return &TestChain{
		bchVirtualChain,
		fakeScanner,
	}
}

func (tc *TestChain) Destroy() {
	tc.Store.Close()
	_ = os.RemoveAll(dbPath)
}

func (tc *TestChain) NewBackend() api.BackendService {
	return api.NewBackend(tc.VirtualChain)
}

func (tc *TestChain) AddTx(txHash gethcmn.Hash, logs ...mevmtypes.Log) {
	tc.scanner.newTxs = append(tc.scanner.newTxs, mevmtypes.Transaction{
		//From: [20]byte{1},
		//To:   [20]byte{2},
		Hash: txHash,
		Logs: logs,
	})
}

func (tc *TestChain) GenNewBlock() (h int64, hash gethcmn.Hash) {
	tc.GenerateNewBlock(true)
	tc.Store.AddBlock(nil)

	return tc.CurrentBlockHeight, tc.CurrentBlockHash
}

func (tc *TestChain) WaitMS(n int64) {
	time.Sleep(time.Duration(n) * time.Millisecond)
}
