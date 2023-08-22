package bch

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchutil"
	"github.com/holiman/uint256"

	"github.com/elfinguard/chainlogs/types"
)

const EGTXFlag = "6a0445475458" //op_return(6a) + len(04) + EGTX(45475458)

func ParseEGTXNullData(script string) (contractAddress [20]byte, topics [][32]byte, otherData [][]byte, err error) {
	if !strings.HasPrefix(script, EGTXFlag) {
		err = types.NotHaveEGTXNulldata
		return
	}
	return ExtractEGTXNullData(strings.TrimPrefix(script, EGTXFlag))
}

func ExtractEGTXNullData(nullDataHexStr string) (contractAddress [20]byte, topics [][32]byte, otherData [][]byte, err error) {
	var script []byte
	script, err = hex.DecodeString(nullDataHexStr)
	if err != nil {
		return
	}
	var es [][]byte
	es, err = txscript.PushedData(script)
	if err != nil {
		return
	}
	for i, e := range es {
		if i == 0 {
			contractAddress = convertToByte20(e)
			continue
		}
		if i >= 1 && i <= 4 {
			if len(e) != 0 {
				topics = append(topics, convertToByte32(e))
			}
		} else {
			otherData = append(otherData, e)
		}
	}
	return
}

func ExtractNullData(nullDataHexStr string) (otherData [][]byte, err error) {
	var script []byte
	script, err = hex.DecodeString(nullDataHexStr)
	if err != nil {
		return
	}
	otherData, err = txscript.PushedData(script)
	return
}

func convertToByte32(o []byte) [32]byte {
	var out [32]byte
	l := len(o)
	if l == 0 {
		return out
	}
	if l <= 32 {
		copy(out[32-l:], o) // pad 0 left
	} else {
		copy(out[:], o[:32]) // trim right
	}
	return out
}

func convertToByte20(o []byte) [20]byte {
	var out [20]byte
	l := len(o)
	if l == 0 {
		return out
	}
	if l <= 20 {
		copy(out[20-l:], o) // pad 0 left
	} else {
		copy(out[:], o[:20]) // trim right
	}
	return out
}

func BuildLogData(confirmations *uint256.Int, outputInfoArray [][32]byte,
	inputInfoArray [][32]byte, outputTokenInfos, inputTokenInfos []TokenInfo, otherDataInOpReturn [][]byte) []byte {
	var outputInfos = make([]*big.Int, len(outputInfoArray))
	for i, output := range outputInfoArray {
		outputInfos[i] = uint256.NewInt(0).SetBytes32(output[:]).ToBig()
	}
	var inputInfos = make([]*big.Int, len(inputInfoArray))
	for i, input := range inputInfoArray {
		inputInfos[i] = uint256.NewInt(0).SetBytes32(input[:]).ToBig()
	}
	return PackEGTXLogDataEvent(confirmations.ToBig(),
		outputInfos, inputInfos, outputTokenInfos, inputTokenInfos, otherDataInOpReturn)
}

func ExtractSenderInfo(originTx *btcjson.TxRawResult, vout uint32, params *chaincfg.Params) (senderInfo [32]byte, err error) {
	originVoutPkScript := originTx.Vout[vout].ScriptPubKey
	originValue := originTx.Vout[vout].Value
	if originVoutPkScript.Type == "pubkeyhash" || originVoutPkScript.Type == "scripthash" {
		if len(originVoutPkScript.Addresses) != 1 {
			err = fmt.Errorf("wrong address count")
			return
		}
		senderAddr, _err := bchutil.DecodeAddress(originVoutPkScript.Addresses[0], params)
		if _err != nil {
			err = fmt.Errorf("failed to decode address: %w", err)
			return
		}
		copy(senderInfo[:20], senderAddr.ScriptAddress())
		amount := uint256.NewInt(0).Mul(uint256.NewInt(uint64(originValue*1e8)), uint256.NewInt(1e10)).Bytes20()
		copy(senderInfo[20:], amount[8:])
		return
	}

	err = fmt.Errorf("invalid pkScript")
	return
}
