package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/rpcclient"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"

	"github.com/elfinguard/chainlogs/bch"
)

type Sender struct {
	mainChainClient *rpcclient.Client

	from bchutil.Address
	wif  *bchutil.WIF
	fee  int64 // miner fee

	contractAddress [20]byte
	payer           [20]byte
	payee           [20]byte
	fileID          [32]byte
	otherData       [32]byte

	payTo  bchutil.Address
	payAmt float64
}

func newSender(mainChainClientInfo, wif string) *Sender {
	s := Sender{
		fee: 400,
	}
	_, s.mainChainClient = bch.MakeMainChainClient(mainChainClientInfo)
	s.initMainChainFields(wif)
	return &s
}

func main() {
	var keyGenFlag bool
	var listUtxoFlag bool
	var dryRunFlag bool
	var mainChainClientInfo string
	var wif string
	var contractEthAddr string
	var payerEthAddr string
	var payeeEthAddr string
	var fileId string
	var payeeBchAddr string
	var payAmt float64
	var minerFee int64

	flag.BoolVar(&keyGenFlag, "key-gen", false, "gen new key")
	flag.BoolVar(&listUtxoFlag, "list-utxo", false, "list UTXO")
	flag.BoolVar(&dryRunFlag, "dry-run", false, "do not send tx")
	flag.StringVar(&mainChainClientInfo, "mainChainClientInfo", "", "bch main chain client info: url,username,password")
	flag.StringVar(&mainChainClientInfo, "rpc", "", "alias of mainChainClientInfo")
	flag.StringVar(&wif, "wif", wif, "main chain wif")
	flag.StringVar(&contractEthAddr, "contract", "0x", "contract address")
	flag.StringVar(&payerEthAddr, "payer", "0x", "payer's ETH address")
	flag.StringVar(&payeeEthAddr, "payee", "0x", "payee's ETH address")
	flag.StringVar(&fileId, "file-id", "0x", "fileID")
	flag.StringVar(&payeeBchAddr, "pay-to", "", "payee's BCH address")
	flag.Float64Var(&payAmt, "pay-amt", 0, "payment amount")
	flag.Int64Var(&minerFee, "miner-fee", 400, "miner fee (in satoshi)")
	flag.Parse()

	if keyGenFlag {
		generateNewKey()
		return
	}

	if len(wif) == 0 {
		fmt.Println("missing WIF")
		flag.Usage()
	}

	s := newSender(mainChainClientInfo, wif)
	copy(s.contractAddress[:], gethcmn.FromHex(contractEthAddr))
	copy(s.payer[:], gethcmn.FromHex(payerEthAddr))
	copy(s.payee[:], gethcmn.FromHex(payeeEthAddr))
	copy(s.fileID[:], gethcmn.FromHex(fileId))
	s.fee = minerFee
	//fmt.Println(minerFee)
	if payeeBchAddr != "" {
		payToAddr, err := bchutil.DecodeAddress(payeeBchAddr, &chaincfg.MainNetParams)
		if err != nil {
			fmt.Println("can not decode pay-to address:", err.Error())
			return
		}
		s.payTo = payToAddr
		s.payAmt = payAmt
	}

	//fmt.Println("cash address  :", s.from.EncodeAddress())
	fmt.Println("script address:", "0x"+hex.EncodeToString(s.from.ScriptAddress()))

	unspentUtxos := s.listUnspentUtxo(s.from)
	if listUtxoFlag {
		j, _ := json.MarshalIndent(unspentUtxos, "", "  ")
		fmt.Println("UTXOs:", string(j))
		return
	}

	for i, unspent := range unspentUtxos {
		_, err := s.buildAndSendEGTX(unspent, dryRunFlag)
		if err != nil {
			fmt.Println("utxo#", i, ":", err)
		} else {
			break
		}
	}
}

func (s *Sender) listUnspentUtxo(address bchutil.Address) []btcjson.ListUnspentResult {
	fmt.Printf("address: %s\n", address.EncodeAddress())
	var unspentList []btcjson.ListUnspentResult
	var err error
	for {
		unspentList, err = s.mainChainClient.ListUnspentMinMaxAddresses(1, 9999, []bchutil.Address{address})
		if err != nil {
			fmt.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("unspent utxos length:%d\n", len(unspentList))
		break
	}
	return unspentList
}

func (s *Sender) buildAndSendEGTX(unspent btcjson.ListUnspentResult, dryRun bool) (*chainhash.Hash, error) {
	tx, err := s.buildEGTX(unspent)
	if err != nil {
		return nil, err
	}
	if dryRun {
		//j, _ := json.MarshalIndent(tx, "", "  ")
		//fmt.Println("tx:", string(j))
		bb := &bytes.Buffer{}
		_ = tx.Serialize(bb)
		fmt.Println("raw tx:", hex.EncodeToString(bb.Bytes()))
		return nil, nil
	}
	return s.sendEGTX(tx)
}

func (s *Sender) buildEGTX(unspent btcjson.ListUnspentResult) (*wire.MsgTx, error) {
	inAmt := int64(unspent.Amount * 1e8)
	payAmt := int64(s.payAmt * 1e8)
	if inAmt <= payAmt+s.fee {
		return nil, errors.New("unspent amount not enough")
	}
	tx := wire.NewMsgTx(2)

	// input
	hash, _ := chainhash.NewHashFromStr(unspent.TxID)
	txIn := wire.NewTxIn(wire.NewOutPoint(hash, unspent.Vout), nil)
	tx.AddTxIn(txIn)

	// op_return output
	nullDataScript := s.buildNullDataScript()
	if len(nullDataScript) != 0 {
		tx.AddTxOut(wire.NewTxOut(0, nullDataScript))
	}

	// pay-to output
	if payAmt > 0 {
		destinationAddrByte, err := txscript.PayToAddrScript(s.payTo)
		if err != nil {
			return nil, err
		}
		txOut := wire.NewTxOut(payAmt, destinationAddrByte)
		tx.AddTxOut(txOut)
	}

	// change
	changeAmt := inAmt - payAmt - s.fee
	if changeAmt > 0 {
		pkScript, err := txscript.PayToAddrScript(s.from)
		if err != nil {
			return nil, err
		}
		txOut := wire.NewTxOut(changeAmt, pkScript)
		tx.AddTxOut(txOut)
	}

	// sign
	scriptPubkey, _ := hex.DecodeString(unspent.ScriptPubKey)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	sigHash, err := txscript.CalcSignatureHash(scriptPubkey, txscript.NewTxSigHashes(tx), hashType, tx, 0, int64(math.Round(unspent.Amount*1e8)), true)
	if err != nil {
		return nil, err
	}
	sig, err := s.wif.PrivKey.SignECDSA(sigHash)
	if err != nil {
		panic(err)
	}
	sigScript, err := txscript.NewScriptBuilder().AddData(append(sig.Serialize(), byte(hashType))).AddData(s.wif.SerializePubKey()).Script()
	if err != nil {
		panic(err)
	}
	tx.TxIn[0].SignatureScript = sigScript
	return tx, nil
}

func (s *Sender) sendEGTX(tx *wire.MsgTx) (*chainhash.Hash, error) {
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	//fmt.Println(hex.EncodeToString(buf.Bytes()))
	txHash, err := s.mainChainClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	out, _ := json.MarshalIndent(tx, "", "  ")
	fmt.Printf("tx: %s\n", string(out))
	fmt.Println("tx hash:", hex.EncodeToString(txHash[:]))
	return txHash, nil
}

func (s *Sender) initMainChainFields(wif string) {
	w, err := bchutil.DecodeWIF(wif)
	if err != nil {
		panic(err)
	}
	s.wif = w
	pkhFrom := bchutil.Hash160(w.SerializePubKey())
	//from, err := bchutil.NewAddressPubKeyHash(pkhFrom, &chaincfg.TestNet3Params)
	from, err := bchutil.NewAddressPubKeyHash(pkhFrom, &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}
	s.from = from

	//err = s.mainChainClient.ImportAddressRescan(from.EncodeAddress(), from.EncodeAddress(), true)
	//if err != nil {
	//	panic(err)
	//}
}

func (s *Sender) buildNullDataScript() []byte {
	//s.contractAddress = [20]byte{0x01}
	//s.payer = [20]byte{0x02}
	//copy(s.payee[:], s.to.ScriptAddress())
	//s.fileID = [32]byte{0x04}
	//s.otherData = [32]byte{0x05}
	script, _ := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte("EGTX")).
		AddData(s.contractAddress[:]).
		AddData(s.payer[:]).      // topic#0
		AddData(s.payee[:]).      // topic#1
		AddData(s.fileID[:]).     // topic#2
		AddOp(txscript.OP_FALSE). // topic#3
		//AddData(s.otherData[:]).
		Script()
	return script
}

func generateNewKey() {
	fmt.Println("generate new key ...")
	params := &chaincfg.MainNetParams
	priv, _ := bchec.NewPrivateKey(bchec.S256())
	wif, err := bchutil.NewWIF(priv, params, false)
	if err != nil {
		panic(err)
	}

	//fmt.Println("network:", params.Name)
	fmt.Println("key WIF:", wif.String())
	pkhFrom := bchutil.Hash160(wif.SerializePubKey())
	from, _ := bchutil.NewAddressPubKeyHash(pkhFrom, params)
	fmt.Println("cash address:", params.CashAddressPrefix+":"+from.EncodeAddress())

	ecdsaKey := (*ecdsa.PrivateKey)(priv)
	evmAddr := gethcrypto.PubkeyToAddress(ecdsaKey.PublicKey)
	fmt.Println("evm address:", evmAddr.String())
}
