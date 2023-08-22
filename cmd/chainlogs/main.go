package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/elfinguard/chainlogs/chains"
	"github.com/elfinguard/chainlogs/config"
	"github.com/elfinguard/chainlogs/rpc"
	"github.com/elfinguard/chainlogs/store"
)

func main() {
	var bchClientInfo string
	flag.StringVar(&bchClientInfo, "bchClientInfo", bchClientInfo, "bch chain client info, format: url,username,password")
	var dbPath string
	flag.StringVar(&dbPath, "dbPath", dbPath, "db path")
	var genesisMainChainBlockHeight int64
	flag.Int64Var(&genesisMainChainBlockHeight, "genesisMainChainBlockHeight", genesisMainChainBlockHeight, "genesis main chain block height which virtual chain scanned from")
	var rpcAddr = "tcp://:8545"
	flag.StringVar(&rpcAddr, "http.addr", rpcAddr, "HTTP-RPC server listening address")
	var wsAddr = "tcp://:8546"
	flag.StringVar(&wsAddr, "ws.addr", wsAddr, "WS-RPC server listening address")
	var rpcAddrSecure = "tcp://:9545"
	flag.StringVar(&rpcAddrSecure, "https.addr", rpcAddrSecure, "HTTPS-RPC server listening address, use special value \"off\" to disable HTTPS")
	var wsAddrSecure = "tcp://:9546"
	flag.StringVar(&wsAddrSecure, "wss.addr", wsAddrSecure, "WSS-RPC server listening address, use special value \"off\" to disable WSS")
	var corsDomain = "*"
	flag.StringVar(&corsDomain, "http.corsdomain", corsDomain, "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	var logLevel = "info"
	flag.StringVar(&logLevel, "logLevel", logLevel, "Log level like tendermint format")
	flag.Parse()

	cfg := config.DefaultConfig()
	bchChainConfig := config.NewBchChainConfig(&cfg, []string{bchClientInfo}, genesisMainChainBlockHeight)
	cfg.RegisterChainConfig(bchChainConfig.ChainName, bchChainConfig)
	defaultRpcEthGetLogsMaxResults := 10000

	logger, err := flags.ParseLogLevel(logLevel, log.NewTMLogger(log.NewSyncWriter(os.Stdout)), "info")
	if err != nil {
		panic(err)
	}
	s := store.NewChainLogDB(dbPath, defaultRpcEthGetLogsMaxResults, logger.With("module", "db"))
	a := chains.NewChainLogs(&cfg, logger.With("module", "adapter"))
	bchVirtualChain := chains.NewBchVirtualChain(bchChainConfig, s, logger.With("module", "vc"))
	a.RegisterChain(bchChainConfig.ChainName, bchVirtualChain)
	rpcServer, err := rpc.NewAndStartServer(bchVirtualChain, dbPath, rpcAddr, wsAddr, rpcAddrSecure, wsAddrSecure, corsDomain, logger.With("module", "rpc"))
	if err != nil {
		panic(err)
	}
	go chains.TrapSignal(func() {
		for _, c := range a.Chains {
			c.Store.Close()
		}
		_ = rpcServer.Stop()
		fmt.Println("exiting...")
	})
	a.Run()
	select {}
}
