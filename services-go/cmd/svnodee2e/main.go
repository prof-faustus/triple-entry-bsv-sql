// svnodee2e funds from the RUNNING SV Node regtest WALLET and broadcasts the hash-chained ECDH-HMAC
// third-entry stream (spendable envelopes, no OP_RETURN/P2SH, SIGHASH_ALL|FORKID) to that real SV Node,
// then cold-rebuilds the ledger from the chain. This validates the system against a genuine funded BSV
// wallet + full-consensus node (stronger than driving raw txs at Teranode), per the operator: use the
// running test wallet, don't ask for funds.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

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
	streamID     = "ledger.acct"
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
func mustHex(s string) []byte { b, _ := hex.DecodeString(s); return b }

type change struct {
	column string
	row    []byte
	op     cc.Op
	value  string
}

var program = []change{
	{"balance", []byte{0x01}, cc.OpInsert, "1000.00"},
	{"balance", []byte{0x01}, cc.OpUpdate, "1500.00"},
	{"balance", []byte{0x02}, cc.OpInsert, "250.00"},
	{"balance", []byte{0x01}, cc.OpUpdate, "1499.95"},
}

func cellKey(row []byte, col string) string { return streamID + "|" + string(row) + "|" + col }

func main() {
	rpc := flag.String("rpc", "http://127.0.0.1:18443", "SV Node RPC URL")
	user := flag.String("user", "cto", "RPC user")
	pass := flag.String("pass", "ctopass", "RPC pass")
	logp := flag.String("log", filepath.Join(os.TempDir(), "te_svnode.log"), "log path")
	flag.Parse()
	if fl, err := os.Create(*logp); err == nil {
		logFile = fl
		defer fl.Close()
	}
	c := node.New(*rpc, *user, *pass)

	bsvPriv, bsvPub := ec.PrivateKeyFromBytes(mustHex(bsvPrivHex))
	addr, err := script.NewAddressFromPublicKey(bsvPub, false)
	ck(err)
	myLock, err := p2pkh.Lock(addr)
	ck(err)
	myLockHex := hex.EncodeToString(myLock.Bytes())
	pkh := []byte(addr.PublicKeyHash)
	wPriv := mustHex(writerMaster)
	cPub := cc.PubFromPriv(mustHex(cpPriv))

	miner, err := c.GetNewAddress()
	ck(err)

	// ---- 1. fund from the SV Node wallet ----
	logf("== funding from SV Node wallet (sendtoaddress) ==")
	fundTxid, err := c.SendToAddress(addr.AddressString, 10.0)
	ck(err)
	_, err = c.GenerateToAddress(1, miner) // confirm
	ck(err)
	vouts, err := c.GetRawTxVerbose(fundTxid)
	ck(err)
	fundVout, fundSats := -1, uint64(0)
	for _, o := range vouts {
		if o.ScriptPubKey.Hex == myLockHex {
			fundVout = o.N
			fundSats = uint64(math.Round(o.Value * 1e8))
			break
		}
	}
	if fundVout < 0 {
		ck(fmt.Errorf("funding tx %s has no output to our key", fundTxid))
	}
	logf("funded from wallet: %s:%d = %d sat", fundTxid, fundVout, fundSats)

	// ---- 2. broadcast the hash-chained stream to the SV Node ----
	logf("== broadcasting hash-chained ECDH-HMAC stream to SV Node ==")
	flagAF := sighash.AllForkID
	prevTxid, prevVout, prevSats := fundTxid, uint32(fundVout), fundSats
	var prevLink []byte
	txids := make([]string, len(program))
	truth := map[string]string{}

	for i, ch := range program {
		m := cc.ChangeMessage{TableID: streamID, RowID: ch.row, ColumnID: ch.column, Op: ch.op, Seq: uint64(i), PrevTxid: prevLink}
		_, gv, err := cc.GeneratorValue(m)
		ck(err)
		cs, err := cc.CommonSecretAsWriter(wPriv, cPub, gv)
		ck(err)
		kh, err := cc.DeriveHMACKey(cs, m)
		ck(err)
		img := cc.ChangeImage(cc.ImagePlaintext, []byte(ch.value), nil)
		tag := cc.Tag(kh, img)
		rec, err := cc.EncodeRecord(cc.FieldRecord{StreamID: []byte(streamID), Message: m, ImageKind: cc.ImagePlaintext, ChangeImage: img, Tag: tag})
		ck(err)
		env, err := bsvscript.BuildEnvelopeIf(rec, pkh)
		ck(err)
		ck(bsvscript.AssertNativeSpendable(env))

		tx := transaction.NewTransaction()
		unlock, err := p2pkh.Unlock(bsvPriv, &flagAF)
		ck(err)
		ck(tx.AddInputFrom(prevTxid, prevVout, myLockHex, prevSats, unlock))
		change := prevSats - envelopeSats
		tx.AddOutput(&transaction.TransactionOutput{Satoshis: envelopeSats, LockingScript: env})
		tx.AddOutput(&transaction.TransactionOutput{Satoshis: change, LockingScript: myLock})
		ck(tx.Sign())
		txid, err := c.SendRawTransaction(tx.Hex())
		ck(err) // SV Node full-consensus acceptance of the spendable-envelope tx
		_, err = c.GenerateToAddress(1, miner)
		ck(err)

		txids[i] = txid
		truth[cellKey(ch.row, ch.column)] = ch.value
		prevTxid, prevVout, prevSats = txid, 1, change
		prevLink = mustHex(txid)
		logf("  seq %d  %s row=%s -> %q  txid=%s (accepted by SV Node)", i, opName(ch.op), string(ch.row), ch.value, txid)
	}

	// ---- 3. cold-rebuild from the SV Node chain + master keys ----
	logf("== cold-rebuild from SV Node chain + keys ==")
	rebuilt, n := coldRebuild(c, txids[len(txids)-1], wPriv, cPub)
	for k, v := range truth {
		if rebuilt[k] != v {
			ck(fmt.Errorf("rebuild mismatch at %s: got %q want %q", k, rebuilt[k], v))
		}
	}
	if len(rebuilt) != len(truth) {
		ck(fmt.Errorf("rebuild size %d != truth %d", len(rebuilt), len(truth)))
	}
	logf("cold-rebuild OK: %d cells across %d chained entries == source", len(rebuilt), n)
	logf("RESULT: SV-NODE WALLET E2E PASS — funded from running wallet, %d third entries accepted by SV Node, cold-rebuild verified", len(program))
}

func coldRebuild(c *node.Client, head string, wPriv, cPub []byte) (map[string]string, int) {
	type ent struct {
		m   cc.ChangeMessage
		img []byte
		tag []byte
	}
	var rev []ent
	txid := head
	for txid != "" {
		rawHex, err := c.GetRawTransaction(txid)
		ck(err)
		tx, err := transaction.NewTransactionFromHex(rawHex)
		ck(err)
		var data []byte
		for _, o := range tx.Outputs {
			if o.LockingScript == nil {
				continue
			}
			if d, err := bsvscript.ExtractEnvelopeData(o.LockingScript); err == nil {
				data = d
				break
			}
		}
		if data == nil {
			ck(fmt.Errorf("no envelope on %s", txid))
		}
		rec, err := cc.DecodeRecord(data)
		ck(err)
		rev = append(rev, ent{rec.Message, rec.ChangeImage, rec.Tag})
		if len(rec.Message.PrevTxid) == 0 {
			txid = ""
		} else {
			txid = hex.EncodeToString(rec.Message.PrevTxid)
		}
	}
	state := map[string]string{}
	for i := len(rev) - 1; i >= 0; i-- {
		e := rev[i]
		_, gv, err := cc.GeneratorValue(e.m)
		ck(err)
		cs, err := cc.CommonSecretAsWriter(wPriv, cPub, gv)
		ck(err)
		kh, err := cc.DeriveHMACKey(cs, e.m)
		ck(err)
		if hex.EncodeToString(cc.Tag(kh, e.img)) != hex.EncodeToString(e.tag) {
			ck(fmt.Errorf("tag verify failed at seq %d", e.m.Seq))
		}
		rd := cc.NewReader(e.img)
		v := rd.Bytes()
		ck(rd.End())
		k := cellKey(e.m.RowID, e.m.ColumnID)
		if e.m.Op == cc.OpDelete {
			delete(state, k)
		} else {
			state[k] = string(v)
		}
	}
	return state, len(rev)
}

func opName(op cc.Op) string {
	switch op {
	case cc.OpInsert:
		return "INSERT"
	case cc.OpUpdate:
		return "UPDATE"
	default:
		return "DELETE"
	}
}
