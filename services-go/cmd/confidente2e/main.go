// confidente2e demonstrates the confidential field path (SYS-HMAC-009) end-to-end on the running
// funded SV Node: a confidential change is recorded on-chain as a BLINDED COMMITMENT (not plaintext);
// the ECDH-HMAC tag still binds the committed value (discoverable/verifiable), the plaintext NEVER
// appears on chain, and a holder of (value, blinding) can open the commitment.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"

	cc "te-bsv/cryptocore"
	"te-bsv/services-go/bsvscript"
	"te-bsv/services-go/node"
)

const (
	bsvPrivHex   = "1122334455667788990011223344556677889900112233445566778899001122"
	writerMaster = "e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262"
	cpPriv       = "f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181"
	stream       = "ledger.hr"
	secretValue  = "SALARY-CONFIDENTIAL-149995" // must never appear on chain
	envelopeSats = 1000
)

var logFile *os.File

func logf(f string, a ...any) {
	s := fmt.Sprintf(f, a...)
	fmt.Println(s)
	if logFile != nil {
		fmt.Fprintln(logFile, s)
		logFile.Sync()
	}
}
func ck(err error) {
	if err != nil {
		logf("RESULT: FAIL: %s", err)
		os.Exit(1)
	}
}
func mh(s string) []byte { b, _ := hex.DecodeString(s); return b }

func main() {
	rpc := flag.String("rpc", "http://127.0.0.1:18443", "SV Node RPC URL")
	user := flag.String("user", "cto", "RPC user")
	pass := flag.String("pass", "ctopass", "RPC pass")
	logp := flag.String("log", filepath.Join(os.TempDir(), "te_confidential.log"), "log path")
	flag.Parse()
	if fl, err := os.Create(*logp); err == nil {
		logFile = fl
		defer fl.Close()
	}
	c := node.New(*rpc, *user, *pass)

	bsvPriv, bsvPub := ec.PrivateKeyFromBytes(mh(bsvPrivHex))
	addr, err := script.NewAddressFromPublicKey(bsvPub, false)
	ck(err)
	myLock, err := p2pkh.Lock(addr)
	ck(err)
	myLockHex := hex.EncodeToString(myLock.Bytes())
	pkh := []byte(addr.PublicKeyHash)
	wPriv := mh(writerMaster)
	cPub := cc.PubFromPriv(mh(cpPriv))

	// fund from the wallet
	miner, err := c.GetNewAddress()
	ck(err)
	fundTxid, err := c.SendToAddress(addr.AddressString, 5.0)
	ck(err)
	_, err = c.GenerateToAddress(1, miner)
	ck(err)
	vouts, err := c.GetRawTxVerbose(fundTxid)
	ck(err)
	fv, fs := -1, uint64(0)
	for _, o := range vouts {
		if o.ScriptPubKey.Hex == myLockHex {
			fv, fs = o.N, uint64(o.Value*1e8+0.5)
			break
		}
	}
	if fv < 0 {
		ck(fmt.Errorf("no funding output"))
	}
	logf("funded from SV Node wallet %s:%d", fundTxid, fv)

	// build a CONFIDENTIAL third entry: on-chain image = commit(value, r), NOT the plaintext
	m := cc.ChangeMessage{TableID: stream, RowID: []byte("emp#1"), ColumnID: "salary", Op: cc.OpInsert, Seq: 0}
	_, gv, err := cc.GeneratorValue(m)
	ck(err)
	cs, err := cc.CommonSecretAsWriter(wPriv, cPub, gv)
	ck(err)
	kh, err := cc.DeriveHMACKey(cs, m)
	ck(err)
	r := make([]byte, 32)
	_, _ = rand.Read(r)
	commitment := cc.Commit([]byte(secretValue), r) // 32-byte blinded commitment
	tag := cc.Tag(kh, commitment)
	rec, err := cc.EncodeRecord(cc.FieldRecord{StreamID: []byte(stream), Message: m, ImageKind: cc.ImageCommitment, ChangeImage: commitment, Tag: tag})
	ck(err)
	env, err := bsvscript.BuildEnvelopeIf(rec, pkh)
	ck(err)

	tx := transaction.NewTransaction()
	flagAF := sighash.AllForkID
	unlock, err := p2pkh.Unlock(bsvPriv, &flagAF)
	ck(err)
	ck(tx.AddInputFrom(fundTxid, uint32(fv), myLockHex, fs, unlock))
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: envelopeSats, LockingScript: env})
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: fs - envelopeSats, LockingScript: myLock})
	ck(tx.Sign())
	txid, err := c.SendRawTransaction(tx.Hex())
	ck(err)
	_, err = c.GenerateToAddress(1, miner)
	ck(err)
	logf("confidential third entry accepted by SV Node: %s", txid)

	// ---- verify the confidentiality boundary from the chain ----
	rawHex, err := c.GetRawTransaction(txid)
	ck(err)
	// (1) the plaintext NEVER appears anywhere in the on-chain transaction
	if strings.Contains(strings.ToLower(rawHex), hex.EncodeToString([]byte(secretValue))) {
		ck(fmt.Errorf("plaintext leaked on chain"))
	}
	logf("  plaintext %q is absent from the on-chain tx ✓ (confidentiality)", secretValue)

	tx2, err := transaction.NewTransactionFromHex(rawHex)
	ck(err)
	var data []byte
	for _, o := range tx2.Outputs {
		if o.LockingScript == nil {
			continue
		}
		if d, e := bsvscript.ExtractEnvelopeData(o.LockingScript); e == nil {
			data = d
			break
		}
	}
	dr, err := cc.DecodeRecord(data)
	ck(err)
	// (2) on-chain image is a 32-byte commitment, not the value
	if dr.ImageKind != cc.ImageCommitment || len(dr.ChangeImage) != 32 {
		ck(fmt.Errorf("on-chain image is not a 32-byte commitment"))
	}
	logf("  on-chain change_image is a 32-byte blinded commitment (image_kind=COMMITMENT) ✓")
	// (3) tag verifies over the commitment from keys alone (discoverable without the value, SYS-HMAC-006/009)
	if hex.EncodeToString(cc.Tag(kh, dr.ChangeImage)) != hex.EncodeToString(dr.Tag) {
		ck(fmt.Errorf("tag does not verify over the on-chain commitment"))
	}
	logf("  ECDH-HMAC tag verifies over the on-chain commitment ✓ (integrity/discoverability)")
	// (4) a holder of (value, r) opens the commitment (binding); a wrong value does not
	if hex.EncodeToString(cc.Commit([]byte(secretValue), r)) != hex.EncodeToString(dr.ChangeImage) {
		ck(fmt.Errorf("commitment does not open to the held value"))
	}
	if hex.EncodeToString(cc.Commit([]byte("SALARY-OTHER"), r)) == hex.EncodeToString(dr.ChangeImage) {
		ck(fmt.Errorf("commitment opened to a wrong value (not binding)"))
	}
	logf("  commitment opens to (value, blinding) and binds it; wrong value rejected ✓")

	logf("RESULT: CONFIDENTIAL PATH PASS (SYS-HMAC-009) — value off-chain, commitment+tag on-chain, accepted by SV Node")
}
