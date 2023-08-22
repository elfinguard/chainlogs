package bch

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var a = MustParseABI(`
[
	{
		"anonymous": true,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "confirmations",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256[]",
				"name": "outputs",
				"type": "uint256[]"
			},
			{
				"indexed": false,
				"internalType": "uint256[]",
				"name": "inputs",
				"type": "uint256[]"
			},
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "addressAndTokenAmount",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "tokenCategory",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentLengthAndHead",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentTail",
						"type": "uint256"
					}
				],
				"internalType": "struct TokenInfo[]",
				"name": "outputTokenInfos",
				"type": "tuple[]"
			},
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "addressAndTokenAmount",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "tokenCategory",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentLengthAndHead",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentTail",
						"type": "uint256"
					}
				],
				"internalType": "struct TokenInfo[]",
				"name": "inputTokenInfos",
				"type": "tuple[]"
			},
			{
				"indexed": false,
				"internalType": "uint8[][]",
				"name": "otherData",
				"type": "bytes[]"
			}
		],
		"name": "EGTXLogData",
		"type": "function"
	}
]
`)

var b = MustParseABI(`
[
	{
		"anonymous": true,
		"inputs": [
			{
				"name": "confirmations",
				"type": "uint256"
			},
			{
				"name": "outputs",
				"type": "uint256[]"
			},
			{
				"name": "inputs",
				"type": "uint256[]"
			},
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "addressAndTokenAmount",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "tokenCategory",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentLengthAndHead",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentTail",
						"type": "uint256"
					}
				],
				"indexed": false,
				"internalType": "struct TokenInfo[]",
				"name": "outputTokenInfos",
				"type": "tuple[]"
			},
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "addressAndTokenAmount",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "tokenCategory",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentLengthAndHead",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "nftCommitmentTail",
						"type": "uint256"
					}
				],
				"indexed": false,
				"internalType": "struct TokenInfo[]",
				"name": "inputTokenInfos",
				"type": "tuple[]"
			},
			{
				"name": "otherData",
				"type": "bytes[]"
			}
		],
		"name": "EGTXLogData",
		"type": "event"
	}
]
`)

func MustParseABI(abiString string) *abi.ABI {
	out, err := abi.JSON(strings.NewReader(abiString))
	if err != nil {
		panic(err)
	}
	return &out
}

func UnPackEGTXLog(data []byte) ([]interface{}, error) {
	return b.Unpack("EGTXLogData", data)
}

type TokenInfo struct {
	AddressAndTokenAmount      *big.Int "json:\"addressAndTokenAmount\""
	TokenCategory              *big.Int "json:\"tokenCategory\""
	NftCommitmentLengthAndHead *big.Int "json:\"nftCommitmentLengthAndHead\""
	NftCommitmentTail          *big.Int "json:\"nftCommitmentTail\""
}

func PackEGTXLogDataEvent(confirmations *big.Int, outputs, inputs []*big.Int, outputTokenInfos, inputTokenInfos []TokenInfo, otherData [][]byte) []byte {
	output, err := a.Pack("EGTXLogData", confirmations, outputs, inputs, outputTokenInfos, inputTokenInfos, otherData)
	if err != nil {
		panic(err)
	}
	return output[4:]
}
