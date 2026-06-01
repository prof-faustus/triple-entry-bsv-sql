// streame2e drives the Phase-2 exit on Teranode regtest (Appendix B.2/B.5/B.6):
//
//  1. fund a controlled key from a matured coinbase;
//  2. build, sign (SIGHASH_ALL|FORKID), and broadcast a hash-chained stream of N
//     ECDH-HMAC third-entry transactions — each carries its record in a SPENDABLE
//     data envelope (no OP_RETURN, no P2SH), spending the prior tx's output so the
//     UTXO lineage IS the stream (prev_txid links M(c));
//  3. discover: recompute each tag from M(c)+keys and match the on-chain record;
//  4. cold-rebuild: from the head txid + master keys alone, walk the chain via
//     prev_txid, verify every tag, and reconstruct the ledger — asserting equality.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
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
	bsvPrivHex    = "1122334455667788990011223344556677889900112233445566778899001122"
	writerMaster  = "e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262"
	counterpartyP = "f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181"
	streamID      = "ledger.acct"
	envelopeSats  = 1000
)

type change struct {
	table, column string
	row           []byte
	op            cc.Op
	value         string
}

var program = []change{
	{"ledger.acct", "balance", []byte{0x01}, cc.OpInsert, "1000.00"},
	{"ledger.acct", "balance", []byte{0x01}, cc.OpUpdate, "1500.00"},
	{"ledger.acct", "balance", []byte{0x02}, cc.OpInsert, "250.00"},
	{"ledger.acct", "balance", []byte{0x01}, cc.OpUpdate, "1499.95"},
}

type rebuiltEntry struct {
	m    cc.ChangeMessage
	img  []byte
	tag  []byte
	kind cc.ImageKind
}

func cellKey(tab string, row []byte, col string) string {
	return tab + "|" + hex.EncodeToString(row) + "|" + col
}

var logFile *os.File

// logf writes to stdout and to a synced log file (local temp), so results survive
// any stdout-capture quirks of the host (D: drive caching, pipe buffering, etc.).
func logf(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	fmt.Println(line)
	if logFile != nil {
		fmt.Fprintln(logFile, line)
		logFile.Sync()
	}
}

func main() {
	url := flag.String("url", "http://localhost:9292", "RPC URL")
	user := flag.String("user", "teranode", "RPC user")
	pass := flag.String("pass", "regtestsecret", "RPC pass")
	logPath := flag.String("log", filepath.Join(os.TempDir(), "te_e2e.log"), "result log path")
	flag.Parse()
	if f, err := os.Create(*logPath); err == nil {
		logFile = f
		defer f.Close()
	}
	c := node.New(*url, *user, *pass)
	logf("log file: %s", *logPath)

	bsvPriv, bsvPub := ec.PrivateKeyFromBytes(mustHex(bsvPrivHex))
	addr, err := script.NewAddressFromPublicKey(bsvPub, false) // regtest uses testnet prefix
	ck(err)
	myLock, err := p2pkh.Lock(addr)
	ck(err)
	myLockHex := hex.EncodeToString(myLock.Bytes())
	pkh := []byte(addr.PublicKeyHash)

	wPriv := mustHex(writerMaster)
	cPub := cc.PubFromPriv(mustHex(counterpartyP))

	// ---- 1. fund from a matured coinbase ----
	logf("== funding ==")
	hashes, err := c.GenerateToAddress(1, addr.AddressString)
	ck(err)
	cbBlock, err := c.GetBlock(hashes[0])
	ck(err)
	_, err = c.Generate(100) // mature coinbase (COINBASE_MATURITY=100)
	ck(err)
	cbHex, err := c.GetRawTransaction(cbBlock.MerkleRoot) // single-tx block: merkleroot == coinbase txid
	ck(err)
	cbTx, err := transaction.NewTransactionFromHex(cbHex)
	ck(err)
	fundVout, fundSats := -1, uint64(0)
	for i, o := range cbTx.Outputs {
		if o.LockingScript != nil && o.LockingScript.Equals(myLock) {
			fundVout, fundSats = i, o.Satoshis
			break
		}
	}
	if fundVout < 0 {
		fail("no coinbase output pays our key")
	}
	logf("funded: %s:%d = %d sat", cbBlock.MerkleRoot, fundVout, fundSats)

	// ---- 2. build + broadcast the hash-chained stream ----
	logf("== broadcasting stream ==")
	flagAF := sighash.AllForkID
	prevTxid, prevVout, prevSats := cbBlock.MerkleRoot, uint32(fundVout), fundSats
	var prevLink []byte // prev_txid for M(c); empty at genesis
	txids := make([]string, len(program))
	truth := map[string]string{}

	for i, ch := range program {
		m := cc.ChangeMessage{TableID: ch.table, RowID: ch.row, ColumnID: ch.column, Op: ch.op, Seq: uint64(i), PrevTxid: prevLink}
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
		if err := bsvscript.AssertNativeSpendable(env); err != nil { // B.5 static check
			fail("envelope not native-spendable: " + err.Error())
		}

		tx := transaction.NewTransaction()
		unlock, err := p2pkh.Unlock(bsvPriv, &flagAF) // SIGHASH_ALL|FORKID (B.6)
		ck(err)
		ck(tx.AddInputFrom(prevTxid, prevVout, myLockHex, prevSats, unlock))
		change := prevSats - envelopeSats // fee 0 (regtest minfee=0)
		tx.AddOutput(&transaction.TransactionOutput{Satoshis: envelopeSats, LockingScript: env})
		tx.AddOutput(&transaction.TransactionOutput{Satoshis: change, LockingScript: myLock})
		ck(tx.Sign())
		txid, err := c.SendRawTransaction(tx.Hex())
		ck(err)
		_, err = c.Generate(1)
		ck(err)

		txids[i] = txid
		truth[cellKey(ch.table, ch.row, ch.column)] = ch.value
		// next tx spends THIS tx's change output (vout 1); M.prev_txid := this txid
		prevTxid, prevVout, prevSats = txid, 1, change
		prevLink = mustHex(txid)
		logf("  seq %d  txid=%s  tag=%x…", i, txid, tag[:6])

		// ---- 3. discovery: recompute tag, match on-chain record ----
		gotHex, err := c.GetRawTransaction(txid)
		ck(err)
		onTx, err := transaction.NewTransactionFromHex(gotHex)
		ck(err)
		recOut := findEnvelope(onTx)
		if recOut == nil {
			fail("no envelope output on broadcast tx")
		}
		data, err := bsvscript.ExtractEnvelopeData(recOut)
		ck(err)
		dr, err := cc.DecodeRecord(data)
		ck(err)
		if hex.EncodeToString(dr.Tag) != hex.EncodeToString(tag) {
			fail("discovery: on-chain tag mismatch")
		}
	}

	// ---- 4. cold-rebuild from head txid + master keys alone ----
	logf("== cold-rebuild from chain + keys ==")
	rebuilt, n, err := coldRebuild(c, txids[len(txids)-1], wPriv, cPub)
	ck(err)
	if n != len(program) {
		fail(fmt.Sprintf("walked %d entries, expected %d", n, len(program)))
	}
	if len(rebuilt) != len(truth) {
		fail(fmt.Sprintf("rebuild size %d != truth %d", len(rebuilt), len(truth)))
	}
	for k, v := range truth {
		if rebuilt[k] != v {
			fail(fmt.Sprintf("rebuild mismatch at %s: got %q want %q", k, rebuilt[k], v))
		}
	}
	logf("cold-rebuild OK: %d ledger cells reconstructed and tag-verified across %d chained entries", len(rebuilt), n)
	logf("RESULT: PHASE-2 E2E PASS — B.2 hash-chain+tags · B.5 spendable, no P2SH/OP_RETURN · B.6 SIGHASH_ALL|FORKID")
}

// coldRebuild walks the stream backward from headTxid via prev_txid, then replays
// forward: verifying each tag from M(c)+keys and applying plaintext changes.
func coldRebuild(c *node.Client, headTxid string, wPriv, cPub []byte) (map[string]string, int, error) {
	var rev []rebuiltEntry
	txid := headTxid
	for txid != "" {
		rawHex, err := c.GetRawTransaction(txid)
		if err != nil {
			return nil, 0, err
		}
		tx, err := transaction.NewTransactionFromHex(rawHex)
		if err != nil {
			return nil, 0, err
		}
		out := findEnvelope(tx)
		if out == nil {
			return nil, 0, fmt.Errorf("no envelope on %s", txid)
		}
		data, err := bsvscript.ExtractEnvelopeData(out)
		if err != nil {
			return nil, 0, err
		}
		rec, err := cc.DecodeRecord(data)
		if err != nil {
			return nil, 0, err
		}
		rev = append(rev, rebuiltEntry{rec.Message, rec.ChangeImage, rec.Tag, rec.ImageKind})
		if len(rec.Message.PrevTxid) == 0 {
			txid = "" // reached genesis
		} else {
			txid = hex.EncodeToString(rec.Message.PrevTxid)
		}
	}

	state := map[string]string{}
	for i := len(rev) - 1; i >= 0; i-- { // genesis -> head
		e := rev[i]
		expectedSeq := uint64(len(rev) - 1 - i)
		if e.m.Seq != expectedSeq {
			return nil, 0, fmt.Errorf("ordering broken: entry has seq %d, expected %d", e.m.Seq, expectedSeq)
		}
		_, gv, err := cc.GeneratorValue(e.m)
		if err != nil {
			return nil, 0, err
		}
		cs, err := cc.CommonSecretAsWriter(wPriv, cPub, gv)
		if err != nil {
			return nil, 0, err
		}
		kh, err := cc.DeriveHMACKey(cs, e.m)
		if err != nil {
			return nil, 0, err
		}
		if hex.EncodeToString(cc.Tag(kh, e.img)) != hex.EncodeToString(e.tag) {
			return nil, 0, fmt.Errorf("tag verification failed at seq %d (tamper/forgery)", e.m.Seq)
		}
		if e.kind == cc.ImagePlaintext {
			val, err := plaintextValue(e.img)
			if err != nil {
				return nil, 0, err
			}
			k := cellKey(e.m.TableID, e.m.RowID, e.m.ColumnID)
			if e.m.Op == cc.OpDelete {
				delete(state, k)
			} else {
				state[k] = val
			}
		}
	}
	return state, len(rev), nil
}

func plaintextValue(img []byte) (string, error) {
	rd := cc.NewReader(img)
	v := rd.Bytes()
	if err := rd.End(); err != nil {
		return "", err
	}
	return string(v), nil
}

func findEnvelope(tx *transaction.Transaction) *script.Script {
	for _, o := range tx.Outputs {
		if o.LockingScript == nil {
			continue
		}
		if _, err := bsvscript.ExtractEnvelopeData(o.LockingScript); err == nil {
			return o.LockingScript
		}
	}
	return nil
}

func mustHex(s string) []byte { b, err := hex.DecodeString(s); ck(err); return b }
func ck(err error) {
	if err != nil {
		fail(err.Error())
	}
}
func fail(msg string) {
	logf("RESULT: FAIL: %s", msg)
	os.Exit(1)
}
