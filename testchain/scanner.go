package testchain

import (
	mdbtypes "github.com/smartbch/moeingdb/types"
	mevmtypes "github.com/smartbch/moeingevm/types"

	"github.com/elfinguard/chainlogs/scanner"
	"github.com/elfinguard/chainlogs/testutils"
)

var _ scanner.IScanner = (*FakeScanner)(nil)

type FakeScanner struct {
	newTxs []mevmtypes.Transaction
}

func (s *FakeScanner) SetLatestScanHeight(blockHeight int64) {

}

func (s *FakeScanner) GetLatestScanHeight() int64 {
	return 0
}

func (s *FakeScanner) GetNewTxs(blockHeight int64, blockHash [32]byte, scanBlock bool) []mdbtypes.Tx {
	newTxs := s.newTxs
	s.newTxs = nil

	mdbTxs := make([]mdbtypes.Tx, len(newTxs))
	for i, mevmTx := range newTxs {
		mevmTx.BlockNumber = blockHeight
		mevmTx.BlockHash = blockHash
		txBytes, _ := mevmTx.MarshalMsg(nil)

		mdbTxs[i] = mdbtypes.Tx{
			HashId:  mevmTx.Hash,
			SrcAddr: mevmTx.From,
			DstAddr: mevmTx.To,
			Content: txBytes,
			LogList: testutils.ToMdbLogs(mevmTx.Logs),
		}
	}

	return mdbTxs
}

func (s *FakeScanner) GetConfirmations(txHash [32]byte) int32 {
	// TODO
	return 0
}
