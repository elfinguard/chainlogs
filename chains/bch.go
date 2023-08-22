package chains

import (
	"github.com/tendermint/tendermint/libs/log"

	"github.com/elfinguard/chainlogs/config"
	"github.com/elfinguard/chainlogs/scanner"
	"github.com/elfinguard/chainlogs/store"
)

func NewBchVirtualChain(cfg *config.ChainConfig, store store.IStore, logger log.Logger) *VirtualChain {
	if len(cfg.ClientUrls) == 0 {
		return nil
	}
	c := VirtualChain{
		Scanner:                     scanner.NewBchScanner(store, cfg.ClientUrls[0], cfg.MaxTxsInBlock, logger.With("module", "scanner")),
		Store:                       store,
		BlockInterval:               cfg.BlockInterval,
		ChainName:                   cfg.ChainName,
		ChainID:                     cfg.ChainId,
		GenesisMainChainBlockHeight: cfg.GenesisMainChainBlockHeight,
		logger:                      logger,
	}
	return &c
}
