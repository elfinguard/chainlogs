package scanner

import (
	modbtypes "github.com/smartbch/moeingdb/types"
)

type IScanner interface {
	GetNewTxs(blockHeight int64, blockHash [32]byte, scanBlock bool) []modbtypes.Tx
	GetConfirmations(txHash [32]byte) int32
	SetLatestScanHeight(blockHeight int64)
	GetLatestScanHeight() int64
}
