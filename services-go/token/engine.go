package token

import (
	"encoding/hex"
	"fmt"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"

	cc "te-bsv/cryptocore"
	"te-bsv/services-go/bsvscript"
	"te-bsv/services-go/node"
)

// Keypair is a controller/holder key with its native P2PKH locking script.
type Keypair struct {
	Priv    *ec.PrivateKey
	Pub     []byte
	Lock    *script.Script
	LockHex string
	Pkh     []byte
}

func NewKeypair(privHex string) (*Keypair, error) {
	pb, err := hex.DecodeString(privHex)
	if err != nil {
		return nil, err
	}
	priv, pub := ec.PrivateKeyFromBytes(pb)
	addr, err := script.NewAddressFromPublicKey(pub, false)
	if err != nil {
		return nil, err
	}
	lock, err := p2pkh.Lock(addr)
	if err != nil {
		return nil, err
	}
	return &Keypair{Priv: priv, Pub: pub.Compressed(), Lock: lock, LockHex: hex.EncodeToString(lock.Bytes()), Pkh: []byte(addr.PublicKeyHash)}, nil
}

// Instance is a live token UTXO (one state of a token's lineage).
type Instance struct {
	Def     Definition
	TokenID string
	Holder  *Keypair
	Value   uint64
	Seq     uint64
	Txid    string
	Vout    uint32
	Sats    uint64
	LockHex string // the envelope locking script of this token UTXO (needed to spend it)
}

// utxo is a spendable output the engine controls (fee wallet).
type utxo struct {
	txid string
	vout uint32
	sats uint64
}

// Engine builds, signs, broadcasts, and journals token lineage events on regtest.
type Engine struct {
	C          *node.Client
	Wallet     *Keypair
	WriterPriv []byte // journalling writer master key
	CpPub      []byte // counterparty/auditor pubkey
	wallet     utxo   // current fee UTXO
}

func stream(typeID string) string { return "token." + typeID }

// Fund mines a coinbase to the wallet key and matures it.
func (e *Engine) Fund() error {
	hashes, err := e.C.GenerateToAddress(1, e.walletAddr())
	if err != nil {
		return err
	}
	blk, err := e.C.GetBlock(hashes[0])
	if err != nil {
		return err
	}
	if _, err := e.C.Generate(100); err != nil {
		return err
	}
	cbHex, err := e.C.GetRawTransaction(blk.MerkleRoot)
	if err != nil {
		return err
	}
	cb, err := transaction.NewTransactionFromHex(cbHex)
	if err != nil {
		return err
	}
	for i, o := range cb.Outputs {
		if o.LockingScript != nil && o.LockingScript.Equals(e.Wallet.Lock) {
			e.wallet = utxo{blk.MerkleRoot, uint32(i), o.Satoshis}
			return nil
		}
	}
	return fmt.Errorf("no coinbase output pays the wallet key")
}

func (e *Engine) walletAddr() string {
	addr, _ := script.NewAddressFromPublicKeyHash(e.Wallet.Pkh, false)
	return addr.AddressString
}

// record builds the third-entry record for a token event and returns (recBytes, tag).
func (e *Engine) record(def Definition, tokenID string, kind EventKind, value uint64, controllerPub []byte, seq uint64, prev []byte) ([]byte, []byte, error) {
	m := cc.ChangeMessage{TableID: stream(def.TypeID), RowID: []byte(tokenID), ColumnID: "state", Op: kind.Op(), Seq: seq, PrevTxid: prev}
	_, gv, err := cc.GeneratorValue(m)
	if err != nil {
		return nil, nil, err
	}
	cs, err := cc.CommonSecretAsWriter(e.WriterPriv, e.CpPub, gv)
	if err != nil {
		return nil, nil, err
	}
	kh, err := cc.DeriveHMACKey(cs, m)
	if err != nil {
		return nil, nil, err
	}
	img := EncodeEvent(def, value, controllerPub, kind)
	tag := cc.Tag(kh, img)
	rec, err := cc.EncodeRecord(cc.FieldRecord{StreamID: []byte(stream(def.TypeID)), Message: m, ImageKind: cc.ImagePlaintext, ChangeImage: img, Tag: tag})
	return rec, tag, err
}

func unlock(k *Keypair) (*p2pkh.P2PKH, error) {
	f := sighash.AllForkID
	return p2pkh.Unlock(k.Priv, &f)
}

// envelopeOut builds a token/event output: spendable envelope (REC) locked to controllerPkh.
func envelopeOut(rec []byte, controllerPkh []byte, sats uint64) (*transaction.TransactionOutput, *script.Script, error) {
	env, err := bsvscript.BuildEnvelopeIf(rec, controllerPkh)
	if err != nil {
		return nil, nil, err
	}
	if err := bsvscript.AssertNativeSpendable(env); err != nil {
		return nil, nil, err
	}
	return &transaction.TransactionOutput{Satoshis: sats, LockingScript: env}, env, nil
}

func (e *Engine) broadcast(tx *transaction.Transaction) (string, error) {
	if err := tx.Sign(); err != nil {
		return "", err
	}
	txid, err := e.C.SendRawTransaction(tx.Hex())
	if err != nil {
		return "", err
	}
	if _, err := e.C.Generate(1); err != nil {
		return "", err
	}
	return txid, nil
}

// Mint creates a new token UTXO held by `holder` (SYS-TOK-001/004, SYS-CASH-002).
func (e *Engine) Mint(def Definition, tokenID string, holder *Keypair, value uint64) (*Instance, error) {
	sats := def.SatoshiQuantity(value)
	rec, _, err := e.record(def, tokenID, Mint, value, holder.Pub, 0, nil)
	if err != nil {
		return nil, err
	}
	out, env, err := envelopeOut(rec, holder.Pkh, sats)
	if err != nil {
		return nil, err
	}
	tx := transaction.NewTransaction()
	u, err := unlock(e.Wallet)
	if err != nil {
		return nil, err
	}
	if err := tx.AddInputFrom(e.wallet.txid, e.wallet.vout, e.Wallet.LockHex, e.wallet.sats, u); err != nil {
		return nil, err
	}
	change := e.wallet.sats - sats // fee 0 (regtest)
	tx.AddOutput(out)
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: change, LockingScript: e.Wallet.Lock})
	txid, err := e.broadcast(tx)
	if err != nil {
		return nil, err
	}
	e.wallet = utxo{txid, 1, change}
	return &Instance{def, tokenID, holder, value, 0, txid, 0, sats, hex.EncodeToString(env.Bytes())}, nil
}

// Transfer spends the token UTXO and creates a successor bound to newHolder (SYS-TOK-004).
func (e *Engine) Transfer(in *Instance, newHolder *Keypair) (*Instance, error) {
	rec, _, err := e.record(in.Def, in.TokenID, Transfer, in.Value, newHolder.Pub, in.Seq+1, mustHex(in.Txid))
	if err != nil {
		return nil, err
	}
	out, env, err := envelopeOut(rec, newHolder.Pkh, in.Sats)
	if err != nil {
		return nil, err
	}
	tx := transaction.NewTransaction()
	uh, err := unlock(in.Holder)
	if err != nil {
		return nil, err
	}
	uw, err := unlock(e.Wallet)
	if err != nil {
		return nil, err
	}
	if err := tx.AddInputFrom(in.Txid, in.Vout, in.LockHex, in.Sats, uh); err != nil {
		return nil, err
	}
	if err := tx.AddInputFrom(e.wallet.txid, e.wallet.vout, e.Wallet.LockHex, e.wallet.sats, uw); err != nil {
		return nil, err
	}
	tx.AddOutput(out)
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: e.wallet.sats, LockingScript: e.Wallet.Lock})
	txid, err := e.broadcast(tx)
	if err != nil {
		return nil, err
	}
	e.wallet = utxo{txid, 1, e.wallet.sats}
	return &Instance{in.Def, in.TokenID, newHolder, in.Value, in.Seq + 1, txid, 0, in.Sats, hex.EncodeToString(env.Bytes())}, nil
}

// Redeem burns the token (issuer-backed) — value returns to the issuer, event journalled (op=DELETE).
func (e *Engine) Redeem(in *Instance, issuer *Keypair) (string, error) {
	rec, _, err := e.record(in.Def, in.TokenID, Redeem, in.Value, issuer.Pub, in.Seq+1, mustHex(in.Txid))
	if err != nil {
		return "", err
	}
	out, _, err := envelopeOut(rec, issuer.Pkh, in.Sats)
	if err != nil {
		return "", err
	}
	tx := transaction.NewTransaction()
	uh, err := unlock(in.Holder)
	if err != nil {
		return "", err
	}
	uw, err := unlock(e.Wallet)
	if err != nil {
		return "", err
	}
	if err := tx.AddInputFrom(in.Txid, in.Vout, in.LockHex, in.Sats, uh); err != nil {
		return "", err
	}
	if err := tx.AddInputFrom(e.wallet.txid, e.wallet.vout, e.Wallet.LockHex, e.wallet.sats, uw); err != nil {
		return "", err
	}
	tx.AddOutput(out)
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: e.wallet.sats, LockingScript: e.Wallet.Lock})
	txid, err := e.broadcast(tx)
	if err != nil {
		return "", err
	}
	e.wallet = utxo{txid, 1, e.wallet.sats}
	return txid, nil
}

// Swap atomically exchanges two token lineages in ONE transaction (deliver-versus-deliver, SYS-TOK-007):
// A's token goes to B and B's token goes to A; both legs journalled. Atomicity = same txid.
func (e *Engine) Swap(a *Instance, b *Instance) (*Instance, *Instance, error) {
	recA, _, err := e.record(a.Def, a.TokenID, Transfer, a.Value, b.Holder.Pub, a.Seq+1, mustHex(a.Txid))
	if err != nil {
		return nil, nil, err
	}
	recB, _, err := e.record(b.Def, b.TokenID, Transfer, b.Value, a.Holder.Pub, b.Seq+1, mustHex(b.Txid))
	if err != nil {
		return nil, nil, err
	}
	outA, envA, err := envelopeOut(recA, b.Holder.Pkh, a.Sats)
	if err != nil {
		return nil, nil, err
	}
	outB, envB, err := envelopeOut(recB, a.Holder.Pkh, b.Sats)
	if err != nil {
		return nil, nil, err
	}
	tx := transaction.NewTransaction()
	ua, err := unlock(a.Holder)
	if err != nil {
		return nil, nil, err
	}
	ub, err := unlock(b.Holder)
	if err != nil {
		return nil, nil, err
	}
	uw, err := unlock(e.Wallet)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.AddInputFrom(a.Txid, a.Vout, a.LockHex, a.Sats, ua); err != nil {
		return nil, nil, err
	}
	if err := tx.AddInputFrom(b.Txid, b.Vout, b.LockHex, b.Sats, ub); err != nil {
		return nil, nil, err
	}
	if err := tx.AddInputFrom(e.wallet.txid, e.wallet.vout, e.Wallet.LockHex, e.wallet.sats, uw); err != nil {
		return nil, nil, err
	}
	tx.AddOutput(outA)
	tx.AddOutput(outB)
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: e.wallet.sats, LockingScript: e.Wallet.Lock})
	txid, err := e.broadcast(tx)
	if err != nil {
		return nil, nil, err
	}
	e.wallet = utxo{txid, 2, e.wallet.sats}
	na := &Instance{a.Def, a.TokenID, b.Holder, a.Value, a.Seq + 1, txid, 0, a.Sats, hex.EncodeToString(envA.Bytes())}
	nb := &Instance{b.Def, b.TokenID, a.Holder, b.Value, b.Seq + 1, txid, 1, b.Sats, hex.EncodeToString(envB.Bytes())}
	return na, nb, nil
}

// LineageState is the cold-rebuilt state of a token lineage.
type LineageState struct {
	Entries       int
	Value         uint64
	ControllerPub []byte
	Alive         bool // false after a redeem (burn)
}

// VerifyLineage walks a token's lineage from headTxid via prev_txid, tag-verifies each event from
// keys, and reconstructs the token's state from the chain alone (SYS-PG-004 analogue for tokens).
// tokenID (= M(c).row) disambiguates txs that carry several token-state records (e.g. an atomic swap).
func (e *Engine) VerifyLineage(headTxid, tokenID string) (LineageState, error) {
	var ls LineageState
	type ent struct {
		m   cc.ChangeMessage
		img []byte
		tag []byte
	}
	var rev []ent
	txid := headTxid
	for txid != "" {
		rawHex, err := e.C.GetRawTransaction(txid)
		if err != nil {
			return ls, err
		}
		tx, err := transaction.NewTransactionFromHex(rawHex)
		if err != nil {
			return ls, err
		}
		var data []byte
		for _, o := range tx.Outputs {
			if o.LockingScript == nil {
				continue
			}
			if d, err := bsvscript.ExtractEnvelopeData(o.LockingScript); err == nil {
				if rec, derr := cc.DecodeRecord(d); derr == nil && rec.Message.ColumnID == "state" && string(rec.Message.RowID) == tokenID {
					data = d
					break
				}
			}
		}
		if data == nil {
			return ls, fmt.Errorf("no token-state record on %s", txid)
		}
		rec, err := cc.DecodeRecord(data)
		if err != nil {
			return ls, err
		}
		rev = append(rev, ent{rec.Message, rec.ChangeImage, rec.Tag})
		if len(rec.Message.PrevTxid) == 0 {
			txid = ""
		} else {
			txid = hex.EncodeToString(rec.Message.PrevTxid)
		}
	}
	ls.Alive = true
	for i := len(rev) - 1; i >= 0; i-- { // genesis -> head
		en := rev[i]
		_, gv, err := cc.GeneratorValue(en.m)
		if err != nil {
			return ls, err
		}
		cs, err := cc.CommonSecretAsWriter(e.WriterPriv, e.CpPub, gv)
		if err != nil {
			return ls, err
		}
		kh, err := cc.DeriveHMACKey(cs, en.m)
		if err != nil {
			return ls, err
		}
		if hex.EncodeToString(cc.Tag(kh, en.img)) != hex.EncodeToString(en.tag) {
			return ls, fmt.Errorf("token tag verify failed at seq %d", en.m.Seq)
		}
		ev, err := DecodeEvent(en.img)
		if err != nil {
			return ls, err
		}
		ls.Value = ev.Value
		ls.ControllerPub = ev.ControllerPub
		if ev.Kind == Redeem {
			ls.Alive = false
		}
	}
	ls.Entries = len(rev)
	return ls, nil
}

func mustHex(s string) []byte { b, _ := hex.DecodeString(s); return b }
