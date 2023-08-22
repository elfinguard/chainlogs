package scanner

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchutil"
	"github.com/holiman/uint256"
	modbtypes "github.com/smartbch/moeingdb/types"
	evmtypes "github.com/smartbch/moeingevm/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/elfinguard/chainlogs/bch"
	"github.com/elfinguard/chainlogs/store"
	"github.com/elfinguard/chainlogs/types"
)

const MaxCacheSize = 100_000

var _ IScanner = &BchScanner{}

type BchScanner struct {
	Client bch.IBchClient
	Store  store.IStore

	MaxTxsInBlock    int
	OriginChainParam *chaincfg.Params

	LatestScanBlockHeight int64

	knownTxCache map[string]struct{} // cache non-EGTXs and mined EGTXs

	logger log.Logger
}

func NewBchScanner(store store.IStore, mainChainClientInfo string, maxTxsInBlock int, logger log.Logger) *BchScanner {
	b := BchScanner{
		Client:           bch.NewRetryableClient(mainChainClientInfo, 10, 999, logger.With("module", "client")),
		Store:            store,
		MaxTxsInBlock:    maxTxsInBlock,
		OriginChainParam: &chaincfg.MainNetParams,
		knownTxCache:     make(map[string]struct{}),
		logger:           logger,
	}
	return &b
}

func (b *BchScanner) SetLatestScanHeight(blockHeight int64) {
	b.LatestScanBlockHeight = blockHeight
}

func (b *BchScanner) GetLatestScanHeight() int64 {
	return b.LatestScanBlockHeight
}

func (b *BchScanner) GetNewTxs(blockHeight int64, blockHash [32]byte, scanBlock bool) []modbtypes.Tx {
	var newModbTxs []modbtypes.Tx
	txIndex := int64(0)
	if scanBlock {
		newModbTxs = b.collectMainChainBlockTxs(blockHeight, blockHash, &txIndex)
	}
	if len(newModbTxs) >= b.MaxTxsInBlock {
		return newModbTxs
	}
	txHashes, err := b.Client.GetRawMempool()
	if err != nil {
		panic(err)
	}
	b.logger.Debug("mempool info", "tx nums", len(txHashes))
	for _, txHash := range txHashes {
		// txHash.String() is the hexadecimal string of the txHash byte-reversed
		txid := txHash.String()
		if b.isKnownTx(txid) {
			continue
		}
		if b.Store.IsTxMined(txid) {
			continue
		}
		tx, err := b.Client.GetRawTransactionVerbose(txHash)
		if err != nil {
			continue
		}
		modbTx, err := b.convertUtxoInfoToTx(tx, txIndex, blockHeight, blockHash)
		if err != nil {
			b.AddKnownTx(txid)
			continue
		}
		newModbTxs = append(newModbTxs, *modbTx)
		b.AddKnownTx(txid)
		if len(newModbTxs) >= b.MaxTxsInBlock {
			break
		}
	}
	return newModbTxs
}

func (b *BchScanner) collectMainChainBlockTxs(blockHeight int64, blockHash [32]byte, txIndex *int64) (newModbTxs []modbtypes.Tx) {
	newestHeight, err := b.Client.GetBlockCount()
	if err != nil {
		panic(err)
	}
	for h := b.LatestScanBlockHeight + 1; h <= newestHeight; h++ {
		hash, err := b.Client.GetBlockHash(h)
		if err != nil {
			panic(err)
		}
		blk, err := b.Client.GetBlockVerboseTx(hash)
		if err != nil {
			panic(err)
		}
		b.logger.Debug("collect main chain block txs", "height", h, "len(txs)", len(blk.Tx))
		for _, tx := range blk.Tx {
			if b.isKnownTx(tx.Txid) {
				continue
			}
			if b.Store.IsTxMined(tx.Txid) {
				continue
			}
			modbTx, err := b.convertUtxoInfoToTx(&tx, *txIndex, blockHeight, blockHash)
			if err != nil {
				// no need to add already mined tx in main chain block
				//b.AddKnownTx(tx.Txid)
				continue
			}
			newModbTxs = append(newModbTxs, *modbTx)
			//b.AddKnownTx(tx.Txid)
			*txIndex++
		}
		b.SetLatestScanHeight(h)
		// allow nums of EGTX bigger than config only in situation which there has more EGTX in current main chain block.
		if len(newModbTxs) >= b.MaxTxsInBlock {
			return
		}
	}
	return
}

func buildTokenInfo(address []byte, tokenData btcjson.TokenDataResult) (bch.TokenInfo, error) {
	var tokenInfo bch.TokenInfo
	var addressAndTokenAmount [32]byte
	copy(addressAndTokenAmount[:20], address[:])
	amount, err := strconv.ParseInt(tokenData.Amount, 10, 64)
	if err != nil {
		return bch.TokenInfo{}, err
	}
	amountBytes := uint256.NewInt(uint64(amount)).Bytes20()
	copy(addressAndTokenAmount[20:], amountBytes[8:])
	tokenInfo.AddressAndTokenAmount = big.NewInt(0).SetBytes(addressAndTokenAmount[:])

	category, err := hex.DecodeString(tokenData.Category)
	if err != nil {
		return bch.TokenInfo{}, err
	}
	if len(category) != 32 {
		return bch.TokenInfo{}, errors.New("invalid category")
	}
	tokenInfo.TokenCategory = big.NewInt(0).SetBytes(category)

	var nftCommitmentLengthAndHead [32]byte
	nftCommitmentLengthAndHead[0] = byte(len(tokenData.Nft.Commitment))
	// none:1, mutable:2, or minting:3
	var capability = 0
	switch tokenData.Nft.Capability {
	case "none":
		capability = 1
		break
	case "mutable":
		capability = 2
		break
	case "minting":
		capability = 3
		break
	}
	nftCommitmentLengthAndHead[1] = byte(capability)
	if len(tokenData.Nft.Commitment) > 40 {
		return bch.TokenInfo{}, errors.New("invalid nft commitment")
	}
	copy(nftCommitmentLengthAndHead[24:], tokenData.Nft.Commitment)
	var nftCommitmentTail [32]byte
	if len(tokenData.Nft.Commitment) > 8 {
		copy(nftCommitmentTail[:], tokenData.Nft.Commitment[8:])
	}
	tokenInfo.NftCommitmentLengthAndHead = big.NewInt(0).SetBytes(nftCommitmentLengthAndHead[:])
	tokenInfo.NftCommitmentTail = big.NewInt(0).SetBytes(nftCommitmentTail[:])
	return tokenInfo, nil
}

func (b *BchScanner) convertUtxoInfoToTx(tx *btcjson.TxRawResult, txIndex, blockHeight int64, blockHash [32]byte) (*modbtypes.Tx, error) {
	var nullData string
	var receiverInfos [][32]byte
	var dstAddr [20]byte
	var srcAddr [20]byte
	var otherNullDatas [][]byte
	var outputTokenInfos []bch.TokenInfo
	for i, vout := range tx.Vout {
		if i == 0 {
			if vout.ScriptPubKey.Type != "nulldata" {
				return nil, types.FirstOutputMustEGTX
			}
			if !strings.HasPrefix(vout.ScriptPubKey.Hex, bch.EGTXFlag) {
				return nil, types.NotHaveEGTXNulldata
			}
			nullData = vout.ScriptPubKey.Hex[len(bch.EGTXFlag):]
		}
		if i == 1 && vout.ScriptPubKey.Type != "pubkeyhash" && vout.ScriptPubKey.Type != "scripthash" {
			return nil, types.SecondOutputInvalid
		}
		if vout.ScriptPubKey.Type == "pubkeyhash" || vout.ScriptPubKey.Type == "scripthash" {
			if len(vout.ScriptPubKey.Addresses) != 1 {
				panic(types.PubkeyScriptAddressNumInvalid.Error())
			}
			var receiverInfo [32]byte
			address, err := bchutil.DecodeAddress(vout.ScriptPubKey.Addresses[0], b.OriginChainParam)
			if err != nil {
				panic(err)
			}
			copy(receiverInfo[:20], address.ScriptAddress())
			amount := uint256.NewInt(0).Mul(uint256.NewInt(uint64(vout.Value*1e8)), uint256.NewInt(1e10)).Bytes20()
			copy(receiverInfo[20:], amount[8:])
			receiverInfos = append(receiverInfos, receiverInfo)
			tokenInfo, err := buildTokenInfo(receiverInfo[:20], vout.TokenData)
			if err != nil {
				// todo: return or not
				fmt.Println("buildTokenInfo err:" + err.Error())
				continue
			}
			outputTokenInfos = append(outputTokenInfos, tokenInfo)
		} else if i > 1 && vout.ScriptPubKey.Type == "nulldata" {
			if len(vout.ScriptPubKey.Hex) > 2 {
				data, err := bch.ExtractNullData(vout.ScriptPubKey.Hex[2:])
				if err == nil {
					otherNullDatas = append(otherNullDatas, data...)
				}
			}
		}
	}
	senderInfos, inputTokenInfos := b.extractInputInfos(tx) // list of [20byte address + 12byte value]
	if len(senderInfos) != 0 {
		copy(srcAddr[:], senderInfos[0][:20])
	}
	if len(receiverInfos) != 0 {
		copy(dstAddr[:], receiverInfos[0][:20])
	}
	contractAddress, topics, otherData, err := bch.ExtractEGTXNullData(nullData)
	if err != nil {
		return nil, err
	}
	otherData = append(otherData, otherNullDatas...)
	data := bch.BuildLogData(uint256.NewInt(0), receiverInfos, senderInfos, outputTokenInfos, inputTokenInfos, otherData)
	txHash := common.HexToHash(tx.Txid)
	modbTx := modbtypes.Tx{
		HashId:  txHash,  //using origin tx Hash
		SrcAddr: srcAddr, // using first p2pkh or p2sh input address
		DstAddr: dstAddr, // using first p2pkh or p2sh output address
		LogList: []modbtypes.Log{
			{
				Address: contractAddress,
				Topics:  topics,
			}},
	}
	evmTx := evmtypes.Transaction{
		Hash:             txHash,
		TransactionIndex: txIndex,
		BlockHash:        blockHash,
		BlockNumber:      blockHeight,
		From:             srcAddr,
		To:               dstAddr,
		Logs: []evmtypes.Log{
			{
				Address:     contractAddress,
				Topics:      topics,
				Data:        data,
				BlockNumber: uint64(blockHeight),
				TxHash:      txHash,
				TxIndex:     uint(txIndex),
				BlockHash:   blockHash,
			}},
	}
	txContent, err := evmTx.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	modbTx.Content = txContent
	b.logger.Debug("new egtx", "txid", tx.Txid, "contract address", common.Address(contractAddress).String())
	return &modbTx, nil
}

func (b *BchScanner) AddKnownTx(txHash string) {
	if len(b.knownTxCache) > MaxCacheSize {
		for tx := range b.knownTxCache { // random evict
			delete(b.knownTxCache, tx)
			break
		}
	}
	b.knownTxCache[txHash] = struct{}{}
}

func (b *BchScanner) isKnownTx(txHash string) bool {
	_, exist := b.knownTxCache[txHash]
	return exist
}

// Getconfirmations return value:
// when tx mined in virtual chain, its finalize number is 0,
// when it mined in source chain the latest block, its finalize number is 1,
// when it not is mined by source chain and is not in mempool either, we think one or more of its inputs have been spent by other tx, so its finalize number is -1.
func (b *BchScanner) GetConfirmations(txHash [32]byte) int32 {
	hash, err := chainhash.NewHash(txHash[:])
	if err != nil {
		panic(err)
	}
	res, err := b.Client.GetRawTransactionVerbose(hash)
	if err != nil {
		return -1
	}
	c := int32(res.Confirmations)
	return c
}

func (b *BchScanner) extractInputInfos(tx *btcjson.TxRawResult) ([][32]byte, []bch.TokenInfo) {
	var senderInfos [][32]byte
	var tokenInfos []bch.TokenInfo
	for _, vin := range tx.Vin {
		txHash, err := chainhash.NewHashFromStr(vin.Txid)
		if err != nil {
			panic(err)
		}
		originTx, err := b.Client.GetRawTransactionVerbose(txHash)
		if err != nil {
			panic(err)
		}
		senderInfo, err := bch.ExtractSenderInfo(originTx, vin.Vout, b.OriginChainParam)
		if err != nil {
			panic(err)
		}
		senderInfos = append(senderInfos, senderInfo)
		tokenInfo, err := buildTokenInfo(senderInfo[:20], originTx.Vout[vin.Vout].TokenData)
		if err != nil {
			// todo: panic or not
			fmt.Println("buildTokenInfo err:" + err.Error())
			continue
		}
		tokenInfos = append(tokenInfos, tokenInfo)
	}
	return senderInfos, tokenInfos
}
