// fundprobe verifies the regtest funding path: mine a coinbase to a key we control,
// then locate the spendable P2PKH output via merkleroot==coinbase-txid + getrawtransaction.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
	"te-bsv/services-go/node"
)

func main() {
	url := flag.String("url", "http://localhost:9292", "Teranode RPC URL")
	user := flag.String("user", "teranode", "RPC user")
	pass := flag.String("pass", "regtestsecret", "RPC pass")
	flag.Parse()

	c := node.New(*url, *user, *pass)
	priv, pub := ec.PrivateKeyFromBytes(mustHex("1122334455667788990011223344556677889900112233445566778899001122"))
	addr, err := script.NewAddressFromPublicKey(pub, false) // regtest uses testnet prefix
	ck(err)
	lock, err := p2pkh.Lock(addr)
	ck(err)
	fmt.Printf("priv-derived address: %s  pkh=%x\n", addr.AddressString, addr.PublicKeyHash)

	hashes, err := c.GenerateToAddress(1, addr.AddressString)
	ck(err)
	fmt.Printf("generatetoaddress -> %v\n", hashes)
	blk, err := c.GetBlock(hashes[0])
	ck(err)
	fmt.Printf("block height=%d num_tx=%d merkleroot=%s\n", blk.Height, blk.NumTx, blk.MerkleRoot)

	// single-tx block => merkleroot == coinbase txid
	rawHex, err := c.GetRawTransaction(blk.MerkleRoot)
	ck(err)
	cb, err := transaction.NewTransactionFromHex(rawHex)
	ck(err)
	fmt.Printf("coinbase %s has %d outputs\n", blk.MerkleRoot, len(cb.Outputs))
	found := -1
	for i, o := range cb.Outputs {
		mine := o.LockingScript != nil && o.LockingScript.Equals(lock)
		fmt.Printf("  vout %d: %d sat  mine=%v\n", i, o.Satoshis, mine)
		if mine {
			found = i
		}
	}
	if found < 0 {
		fmt.Println("RESULT: no coinbase output pays our key — generatetoaddress did not fund us")
		os.Exit(1)
	}
	fmt.Printf("RESULT: spendable funding UTXO = %s:%d (%d sat)\n", blk.MerkleRoot, found, cb.Outputs[found].Satoshis)
	_ = priv
}

func mustHex(s string) []byte { b, err := hex.DecodeString(s); ck(err); return b }
func ck(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
