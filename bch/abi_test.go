package bch

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestABI(t *testing.T) {
	confirmations := big.NewInt(1)
	outputs := []*big.Int{big.NewInt(3), big.NewInt(4)}
	inputs := []*big.Int{big.NewInt(5), big.NewInt(6)}
	category, err := hex.DecodeString("0f3dd8905e42c5911245da88ca3600bb4f39d9cbcad7672512e59d059b2428b6")
	outputTokenInfos := []TokenInfo{
		{
			AddressAndTokenAmount:      big.NewInt(100),
			TokenCategory:              big.NewInt(0).SetBytes(category),
			NftCommitmentLengthAndHead: big.NewInt(1),
			NftCommitmentTail:          big.NewInt(10),
		},
	}

	otherData := [][]byte{{7}}
	res, err := UnPackEGTXLog(PackEGTXLogDataEvent(confirmations, outputs, inputs, outputTokenInfos, nil, otherData))
	require.Nil(t, err)
	require.Equal(t, 6, len(res))
	fmt.Println(reflect.TypeOf(res[3]))
	tokenInfos, ok := res[3].([]struct {
		AddressAndTokenAmount      *big.Int "json:\"addressAndTokenAmount\""
		TokenCategory              *big.Int "json:\"tokenCategory\""
		NftCommitmentLengthAndHead *big.Int "json:\"nftCommitmentLengthAndHead\""
		NftCommitmentTail          *big.Int "json:\"nftCommitmentTail\""
	})
	require.True(t, ok)
	require.True(t, bytes.Equal(category, tokenInfos[0].TokenCategory.Bytes()))
	otherDataInRes, ok := res[5].([][]byte)
	require.True(t, ok)
	require.Equal(t, uint8(7), otherDataInRes[0][0])
}
