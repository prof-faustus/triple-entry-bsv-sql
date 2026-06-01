// tewriter is the Phase-3 journalling writer + cold-rebuild verifier.
//
// It drains te.outbox (changes captured atomically by the SQL trigger), and for each change:
//   builds M(c) -> GV -> CS -> K_hmac -> tag, encodes the self-describing record, wraps it in a
//   SPENDABLE envelope (no OP_RETURN/P2SH), and broadcasts it as a hash-chained third entry on
//   Teranode regtest (SIGHASH_ALL|FORKID), writing the txid back to te.outbox + te.chain_index.
// Then it cold-rebuilds the journalled table from the chain + master keys alone and asserts the
// reconstruction equals the live database (SYS-PG-004 / Appendix B.2-B.6).
package main

import (
	"context"
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
	"github.com/jackc/pgx/v5"

	cc "te-bsv/cryptocore"
	"te-bsv/services-go/bsvscript"
	"te-bsv/services-go/node"
)

const (
	bsvPrivHex   = "1122334455667788990011223344556677889900112233445566778899001122"
	envelopeSats = 1000
)

var logFile *os.File

func logf(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	fmt.Println(line)
	if logFile != nil {
		fmt.Fprintln(logFile, line)
		logFile.Sync()
	}
}

type relationship struct {
	writerPriv []byte
	cpPub      []byte
}

type outboxRow struct {
	seq       int64
	stream    string
	table     string
	rowID     []byte
	column    string
	op        int16
	value     *string
}

func cellKey(tab string, row []byte, col string) string {
	return tab + "|" + string(row) + "|" + col
}

func main() {
	rpcURL := flag.String("rpc", "http://localhost:9292", "Teranode RPC URL")
	rpcUser := flag.String("user", "teranode", "RPC user")
	rpcPass := flag.String("pass", "regtestsecret", "RPC pass")
	pg := flag.String("pg", "postgres://te:te@127.0.0.1:5433/te", "PostgreSQL DSN")
	logPath := flag.String("log", filepath.Join(os.TempDir(), "te_writer.log"), "log path")
	flag.Parse()
	if f, err := os.Create(*logPath); err == nil {
		logFile = f
		defer f.Close()
	}
	ctx := context.Background()
	c := node.New(*rpcURL, *rpcUser, *rpcPass)

	db, err := pgx.Connect(ctx, *pg)
	ck(err)
	defer db.Close(ctx)
	logf("connected: PostgreSQL + Teranode regtest")

	// load relationships (keys per stream)
	rels := map[string]relationship{}
	rows, err := db.Query(ctx, `SELECT stream_id, writer_priv, counterparty_pub FROM te.relationship`)
	ck(err)
	for rows.Next() {
		var s string
		var wp, cp []byte
		ck(rows.Scan(&s, &wp, &cp))
		rels[s] = relationship{wp, cp}
	}
	rows.Close()

	// fund a controlled key from a matured coinbase
	bsvPriv, bsvPub := ec.PrivateKeyFromBytes(mustHex(bsvPrivHex))
	addr, err := script.NewAddressFromPublicKey(bsvPub, false)
	ck(err)
	myLock, err := p2pkh.Lock(addr)
	ck(err)
	myLockHex := hex.EncodeToString(myLock.Bytes())
	pkh := []byte(addr.PublicKeyHash)
	utxoTxid, utxoVout, utxoSats := fund(c, addr.AddressString, myLock)
	logf("funded %s:%d = %d sat", utxoTxid, utxoVout, utxoSats)

	// drain the outbox in commit order
	streamSeq := map[string]uint64{}
	prevLink := map[string][]byte{}
	flagAF := sighash.AllForkID

	obrows, err := db.Query(ctx, `SELECT seq, stream_id, table_name, row_id, column_id, op, value
	                              FROM te.outbox WHERE status='pending' ORDER BY seq`)
	ck(err)
	var pending []outboxRow
	for obrows.Next() {
		var r outboxRow
		ck(obrows.Scan(&r.seq, &r.stream, &r.table, &r.rowID, &r.column, &r.op, &r.value))
		pending = append(pending, r)
	}
	obrows.Close()
	logf("draining %d pending changes", len(pending))

	for _, r := range pending {
		rel, ok := rels[r.stream]
		if !ok {
			fail("no relationship/keys for stream " + r.stream)
		}
		sseq := streamSeq[r.stream]
		m := cc.ChangeMessage{TableID: r.table, RowID: r.rowID, ColumnID: r.column, Op: cc.Op(r.op), Seq: sseq, PrevTxid: prevLink[r.stream]}
		_, gv, err := cc.GeneratorValue(m)
		ck(err)
		cs, err := cc.CommonSecretAsWriter(rel.writerPriv, rel.cpPub, gv)
		ck(err)
		kh, err := cc.DeriveHMACKey(cs, m)
		ck(err)
		val := ""
		if r.value != nil {
			val = *r.value
		}
		img := cc.ChangeImage(cc.ImagePlaintext, []byte(val), nil)
		tag := cc.Tag(kh, img)
		rec, err := cc.EncodeRecord(cc.FieldRecord{StreamID: []byte(r.stream), Message: m, ImageKind: cc.ImagePlaintext, ChangeImage: img, Tag: tag})
		ck(err)
		env, err := bsvscript.BuildEnvelopeIf(rec, pkh)
		ck(err)
		if err := bsvscript.AssertNativeSpendable(env); err != nil {
			fail("envelope: " + err.Error())
		}

		tx := transaction.NewTransaction()
		unlock, err := p2pkh.Unlock(bsvPriv, &flagAF)
		ck(err)
		ck(tx.AddInputFrom(utxoTxid, utxoVout, myLockHex, utxoSats, unlock))
		change := utxoSats - envelopeSats
		tx.AddOutput(&transaction.TransactionOutput{Satoshis: envelopeSats, LockingScript: env})
		tx.AddOutput(&transaction.TransactionOutput{Satoshis: change, LockingScript: myLock})
		ck(tx.Sign())
		txid, err := c.SendRawTransaction(tx.Hex())
		ck(err)
		_, err = c.Generate(1)
		ck(err)

		txidBytes := mustHex(txid)
		_, err = db.Exec(ctx, `UPDATE te.outbox SET status='sent', stream_seq=$1, txid=$2 WHERE seq=$3`, int64(sseq), txidBytes, r.seq)
		ck(err)
		_, err = db.Exec(ctx, `INSERT INTO te.chain_index(stream_id, seq, txid) VALUES($1,$2,$3)
		                       ON CONFLICT (stream_id,seq) DO UPDATE SET txid=EXCLUDED.txid`, r.stream, int64(sseq), txidBytes)
		ck(err)

		logf("  %s seq %d  %s row=%s %s -> %q  txid=%s", r.stream, sseq, opName(r.op), string(r.rowID), r.column, val, txid)
		streamSeq[r.stream] = sseq + 1
		prevLink[r.stream] = txidBytes
		utxoTxid, utxoVout, utxoSats = txid, 1, change
	}

	// cold-rebuild each stream from its chain head + keys, compare to the live DB
	logf("== cold-rebuild from chain + keys, compare to live DB ==")
	for stream, rel := range rels {
		var headTxid []byte
		err := db.QueryRow(ctx, `SELECT txid FROM te.chain_index WHERE stream_id=$1 ORDER BY seq DESC LIMIT 1`, stream).Scan(&headTxid)
		if err == pgx.ErrNoRows {
			continue
		}
		ck(err)
		rebuilt, n, err := coldRebuild(c, hex.EncodeToString(headTxid), rel.writerPriv, rel.cpPub)
		ck(err)
		logf("  stream %s: %d entries verified, %d cells rebuilt", stream, n, len(rebuilt))
		// project the live DB to cells and compare
		live := liveCells(ctx, db, stream)
		if len(live) != len(rebuilt) {
			fail(fmt.Sprintf("stream %s: rebuilt %d cells != live %d", stream, len(rebuilt), len(live)))
		}
		for k, v := range live {
			if rebuilt[k] != v {
				fail(fmt.Sprintf("stream %s: mismatch at %s: rebuilt %q live %q", stream, k, rebuilt[k], v))
			}
		}
		logf("  stream %s: cold-rebuild == live DB ✓", stream)
	}
	logf("RESULT: PHASE-3 E2E PASS — ordinary SQL -> verifiable on-chain third entries -> cold-rebuild == live DB")
}

// liveCells projects journalled tables of a stream into (table|row|column)->value cells.
func liveCells(ctx context.Context, db *pgx.Conn, stream string) map[string]string {
	out := map[string]string{}
	jr, err := db.Query(ctx, `SELECT table_name, pk_columns FROM te.journalled WHERE stream_id=$1`, stream)
	ck(err)
	type jt struct {
		name string
		pk   []string
	}
	var tables []jt
	for jr.Next() {
		var t jt
		ck(jr.Scan(&t.name, &t.pk))
		tables = append(tables, t)
	}
	jr.Close()
	for _, t := range tables {
		// demo schema: public.accounts(id, balance); generic projection of non-pk text columns
		rows, err := db.Query(ctx, fmt.Sprintf(`SELECT id::text, balance FROM %s`, t.name))
		ck(err)
		for rows.Next() {
			var id, bal string
			ck(rows.Scan(&id, &bal))
			out[cellKey(t.name, []byte(id), "balance")] = bal
		}
		rows.Close()
	}
	return out
}

func coldRebuild(c *node.Client, headTxid string, wPriv, cPub []byte) (map[string]string, int, error) {
	type entry struct {
		m   cc.ChangeMessage
		img []byte
		tag []byte
	}
	var rev []entry
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
			return nil, 0, fmt.Errorf("no envelope on %s", txid)
		}
		rec, err := cc.DecodeRecord(data)
		if err != nil {
			return nil, 0, err
		}
		rev = append(rev, entry{rec.Message, rec.ChangeImage, rec.Tag})
		if len(rec.Message.PrevTxid) == 0 {
			txid = ""
		} else {
			txid = hex.EncodeToString(rec.Message.PrevTxid)
		}
	}
	state := map[string]string{}
	for i := len(rev) - 1; i >= 0; i-- { // genesis -> head
		e := rev[i]
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
			return nil, 0, fmt.Errorf("tag verify failed at seq %d", e.m.Seq)
		}
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
	return state, len(rev), nil
}

// fund mines a coinbase to addr, matures it, and returns the spendable (txid,vout,sats).
func fund(c *node.Client, addrStr string, myLock *script.Script) (string, uint32, uint64) {
	hashes, err := c.GenerateToAddress(1, addrStr)
	ck(err)
	blk, err := c.GetBlock(hashes[0])
	ck(err)
	_, err = c.Generate(100)
	ck(err)
	cbHex, err := c.GetRawTransaction(blk.MerkleRoot)
	ck(err)
	cb, err := transaction.NewTransactionFromHex(cbHex)
	ck(err)
	for i, o := range cb.Outputs {
		if o.LockingScript != nil && o.LockingScript.Equals(myLock) {
			return blk.MerkleRoot, uint32(i), o.Satoshis
		}
	}
	fail("no coinbase output pays our key")
	return "", 0, 0
}

func plaintextValue(img []byte) (string, error) {
	rd := cc.NewReader(img)
	v := rd.Bytes()
	if err := rd.End(); err != nil {
		return "", err
	}
	return string(v), nil
}

func opName(op int16) string {
	switch cc.Op(op) {
	case cc.OpInsert:
		return "INSERT"
	case cc.OpUpdate:
		return "UPDATE"
	case cc.OpDelete:
		return "DELETE"
	}
	return "?"
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
