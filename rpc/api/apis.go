package api

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/elfinguard/chainlogs/api"
	"github.com/elfinguard/chainlogs/rpc/api/filters"
)

const (
	namespaceEth = "eth"
	apiVersion   = "1.0"
)

// GetAPIs returns the list of all APIs from the Ethereum namespaces
func GetAPIs(backend api.BackendService, logger log.Logger) []rpc.API {
	logger = logger.With("module", "json-rpc")
	_ethAPI := newEthAPI(backend, logger)
	_filterAPI := filters.NewAPI(backend, logger)

	return []rpc.API{
		{
			Namespace: namespaceEth,
			Version:   apiVersion,
			Service:   _ethAPI,
			Public:    true,
		},
		{
			Namespace: namespaceEth,
			Version:   apiVersion,
			Service:   _filterAPI,
			Public:    true,
		},
	}
}
