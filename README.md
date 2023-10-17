# Chainlogs

Chainlogs is UTXO Chains' adaptor to ElfinGuard Authorizer.

UTXO-based blockchains, such as Bitcoin, Bitcoin Cash, Litecoin, Dogecoin, do not have a logging scheme like EVM. Chainlogs can derive an EVM transaction containing one EVM log from a UTXO-based transaction, such that ElfinGuard's authorizers can know what happened on these chains.

The deriving rules are as follows:

1. The first output must be an OP\_RETURN output whose first data element is "EGTX"; otherwise, this transaction is NOT a derivable transaction. If there are multiple OP\_RETURN outputs, all of their pushed data elements will be collected.

2. The second data element of OP\_RETURN are mapped to the EVM Log's source contract address. If it is less than 20 bytes, it will be zero-padded. If it is longer than 20 bytes, it will be tail-truncated.

3. The 3rd~6th non-emtpy data elements are mapped to LOG1/LOG2/LOG3/LOG4 instructions' topics, i.e., the solidity events' topics. Each one will be coverted to a big-endian bytes32 by zero-padding and tail-truncation. Empty data elements pushed by OP\_FALSE are ignored and not mapped as topics.

4. The following data are concatenated together to form the LOG instructions' data. 

   1. The confirmation count as a uint256. Yes, this number can be changed as time goes by. It is not a constant number. It will be -1 if the transaction cannot be found in mempool or the block history any more.

   2. The P2PKH/P2SH outputs' recipient address (20 bytes) and value (12 bytes) as a uint256 array (`uint256[]`). The value is multiplied by `10**10` such that its unit is changed from satoshi to wei.

   3. The P2PKH/P2SH inputs' sender address (20 bytes) and value (12 bytes) as a uint256 array (`uint256[]`). The value is multiplied by `10**10` such that its unit is changed from satoshi to wei.

   4. The P2PKH/P2SH outputs' token information as an array (`TokenInfo[]`).

   5. The P2PKH/P2SH inputs' token information as an array (`TokenInfo[]`).

   6. The other data elements from all the OP\_RETURN outputs as a bytes array (`bytes[]`).

5. The first P2PKH/P2SH outputs' recipient address is taken as the to-address of the EVM transaction

6. The first P2PKH/P2SH inputs' sender address is taken as the from-address of the EVM transaction

The `TokenInfo` struct is defined as:

```solidty
struct TokenInfo {
    uint256 addressAndTokenAmount; // The P2PKH/P2SH address (20 bytes) and token amount (12 bytes)
    uint256 tokenCategory;
    uint256 nftCommitmentLengthAndHead; // The length and leading 8 bytes of nftCommitment
    uint256 nftCommitmentTail; // The trailing 32 bytes of nftCommitment
}
```

Currently, we only implement an adaptor for Bitcoin Cash in this repo.

We use virtual block number for these EVM logs. Different UTXO Adapter may generate different blocks for the same block number. The mempool is checked every 5 seconds and any new derivable transactions will be packed to a new virtual block. The mined blocks are also checked to find new derivable transactions.

The name of these blockchains (Bitcoin, Bitcoin Cash, Litecoin, Dogecoin) are prefixed with "virtual" and then mapped to bytes32 as their EVM chainId.

It is recommended that the source contract address (20 bytes) is calculated as `RIPEMD160(SHA256(URI))`. The URI is controlled by the authorizing contract's developers.

It is recommended that the derived logs are viewed as solidity's anonymous events, because always attaching the same 32 bytes to OP\_RETURN is a waste.

The virtual EVM blocks' attributes are left empty or zero except two:

1. Size: it is reused to store the time when this virtual block is build.

2. GasUsed: it is reused to store the main chain's height scanned by this adaptor when this virtual block is build.

These attributes depends on the local time of the adaptor's machine and the timing it sees the mempool's transactions. So they will differ from one adaptor to another adaptor.

We don't want authorizers disagree on timestamp. So the local timestamp is store in 'Size' which is ignored by authorizers. And the blocks' timestamps are left as zero.
