package types

import "errors"

var (
	FirstOutputMustEGTX           = errors.New("first output must EGTX typed nulldata")
	SecondOutputInvalid           = errors.New("second output must P2PKH/P2SH")
	NotHaveEGTXNulldata           = errors.New("not have EGTX typed nulldata")
	NotHaveContractAddress        = errors.New("not have contract address")
	PubkeyScriptAddressNumInvalid = errors.New("invalid pubkey script address num")
)
