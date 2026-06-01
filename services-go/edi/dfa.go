// Package edi implements the commercial-document DFA engine (Phase 5), grounded in US20220253835A1,
// native BSV. Each document lifecycle is a deterministic finite automaton whose states are UTXOs: a
// transition spends the current state UTXO and creates the next (SYS-EDI-001), journalled as a third
// entry (SYS-EDI-003). Document types and their transition tables are data (SYS-EDI-002); the engine
// runs any of them. Cross-references by object_id (SYS-EDI-004). The same engine drives the logistics
// consignment lifecycle, with B/L-as-token title transfer via controller re-keying (SYS-LOG-006).
package edi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"

	cc "te-bsv/cryptocore"
	"te-bsv/services-go/bsvscript"
	"te-bsv/services-go/node"
)

// Transition is one DFA edge.
type Transition struct {
	From  string `json:"from"`
	Event string `json:"event"`
	To    string `json:"to"`
}

// DFADef is a document type's automaton (SYS-EDI-002): state set, event alphabet, transition table.
type DFADef struct {
	DocType     string       `json:"doc_type"`
	Initial     string       `json:"initial"`
	Final       []string     `json:"final"`
	Transitions []Transition `json:"transitions"`
}

func (d DFADef) next(from, event string) (string, bool) {
	for _, t := range d.Transitions {
		if t.From == from && t.Event == event {
			return t.To, true
		}
	}
	return "", false
}

func (d DFADef) isFinal(s string) bool {
	for _, f := range d.Final {
		if f == s {
			return true
		}
	}
	return false
}

// IsFinal reports whether s is an accepting/final state.
func (d DFADef) IsFinal(s string) bool { return d.isFinal(s) }

// From returns the transitions available from state s.
func (d DFADef) From(s string) []Transition {
	var out []Transition
	for _, t := range d.Transitions {
		if t.From == s {
			out = append(out, t)
		}
	}
	return out
}

type Registry map[string]DFADef

func LoadRegistry(path string) (Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var defs []DFADef
	if err := json.Unmarshal(b, &defs); err != nil {
		return nil, err
	}
	r := Registry{}
	for _, d := range defs {
		r[d.DocType] = d
	}
	return r, nil
}

func (r Registry) Get(t string) (DFADef, error) {
	d, ok := r[t]
	if !ok {
		return DFADef{}, fmt.Errorf("unknown doc type %q", t)
	}
	return d, nil
}

// Keypair is a controller key with its native P2PKH script.
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
	return &Keypair{priv, pub.Compressed(), lock, hex.EncodeToString(lock.Bytes()), []byte(addr.PublicKeyHash)}, nil
}

// Doc is a live document/consignment state (one UTXO of its DFA lineage).
type Doc struct {
	DocType    string
	ObjectID   string
	State      string
	Refs       []string // cross-referenced object_ids (SYS-EDI-004)
	BodyHashH4 []byte   // on-chain anchor of the off-chain body (GB2558485A)
	Controller *Keypair
	Seq        uint64
	Txid       string
	Vout       uint32
	Sats       uint64
	LockHex    string
}

// payload is the DFA-state change_image committed by the third-entry tag.
func encodePayload(d *Doc) []byte {
	w := cc.NewWriter().Str(d.DocType).Str(d.ObjectID).Str(d.State).U32(uint32(len(d.Refs)))
	for _, r := range d.Refs {
		w.Str(r)
	}
	return w.Bytes(d.BodyHashH4).Bytes(d.Controller.Pub).Finish()
}

// DecodedState is a cold-read of a state payload.
type DecodedState struct {
	DocType, ObjectID, State string
	Refs                     []string
	BodyHashH4               []byte
	ControllerPub            []byte
}

func decodePayload(b []byte) (DecodedState, error) {
	var s DecodedState
	rd := cc.NewReader(b)
	s.DocType = rd.Str()
	s.ObjectID = rd.Str()
	s.State = rd.Str()
	n := rd.U32()
	for i := uint32(0); i < n; i++ {
		s.Refs = append(s.Refs, rd.Str())
	}
	s.BodyHashH4 = rd.Bytes()
	s.ControllerPub = rd.Bytes()
	return s, rd.End()
}

type utxo struct {
	txid string
	vout uint32
	sats uint64
}

const stateSats = 1000

// Engine builds/signs/broadcasts/journals DFA transitions on regtest.
type Engine struct {
	C          *node.Client
	Wallet     *Keypair
	WriterPriv []byte
	CpPub      []byte
	Reg        Registry
	MinerAddr  string // if set, mine via generatetoaddress (SV Node); else generate (Teranode)
	wallet     utxo
	lastBlock  string // hash of the block mined by the most recent send() (reliable tip ref)
}

// LastBlockHash returns the block hash mined by the most recent transition (generate-returned, so it
// reflects the true tip even when getblockchaininfo/getbestblockhash lag on Teranode).
func (e *Engine) LastBlockHash() string { return e.lastBlock }

// mine confirms a tx and records the mined block hash. generatetoaddress on a wallet node (SV Node),
// else generate (Teranode regtest).
func (e *Engine) mine() error {
	var hashes []string
	var err error
	if e.MinerAddr != "" {
		hashes, err = e.C.GenerateToAddress(1, e.MinerAddr)
	} else {
		hashes, err = e.C.Generate(1)
	}
	if err == nil && len(hashes) > 0 {
		e.lastBlock = hashes[len(hashes)-1]
	}
	return err
}

// FundFromWallet funds the fee key from a node wallet (SV Node) via sendtoaddress.
func (e *Engine) FundFromWallet(amountBSV float64) error {
	txid, err := e.C.SendToAddress(e.Wallet.addr(), amountBSV)
	if err != nil {
		return err
	}
	if err := e.mine(); err != nil {
		return err
	}
	vouts, err := e.C.GetRawTxVerbose(txid)
	if err != nil {
		return err
	}
	want := hex.EncodeToString(e.Wallet.Lock.Bytes())
	for _, o := range vouts {
		if o.ScriptPubKey.Hex == want {
			e.wallet = utxo{txid, uint32(o.N), uint64(o.Value*1e8 + 0.5)}
			return nil
		}
	}
	return fmt.Errorf("funding tx %s has no output to the wallet key", txid)
}

func (e *Engine) Fund() error {
	hashes, err := e.C.GenerateToAddress(1, e.Wallet.addr())
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
	return fmt.Errorf("coinbase did not pay wallet")
}

func (k *Keypair) addr() string {
	a, _ := script.NewAddressFromPublicKeyHash(k.Pkh, false)
	return a.AddressString
}

func (e *Engine) unlock(k *Keypair) (*p2pkh.P2PKH, error) {
	f := sighash.AllForkID
	return p2pkh.Unlock(k.Priv, &f)
}

// recAndEnvelope builds the journalled record + spendable envelope for a doc state.
func (e *Engine) recAndEnvelope(d *Doc, op cc.Op, prev []byte) (*script.Script, error) {
	m := cc.ChangeMessage{TableID: d.DocType, RowID: []byte(d.ObjectID), ColumnID: "state", Op: op, Seq: d.Seq, PrevTxid: prev}
	_, gv, err := cc.GeneratorValue(m)
	if err != nil {
		return nil, err
	}
	cs, err := cc.CommonSecretAsWriter(e.WriterPriv, e.CpPub, gv)
	if err != nil {
		return nil, err
	}
	kh, err := cc.DeriveHMACKey(cs, m)
	if err != nil {
		return nil, err
	}
	img := encodePayload(d)
	tag := cc.Tag(kh, img)
	rec, err := cc.EncodeRecord(cc.FieldRecord{StreamID: []byte(d.DocType), Message: m, ImageKind: cc.ImagePlaintext, ChangeImage: img, Tag: tag})
	if err != nil {
		return nil, err
	}
	env, err := bsvscript.BuildEnvelopeIf(rec, d.Controller.Pkh)
	if err != nil {
		return nil, err
	}
	if err := bsvscript.AssertNativeSpendable(env); err != nil {
		return nil, err
	}
	return env, nil
}

// Originate creates the genesis state UTXO of a document/consignment (US20220253835A1 origination tx).
func (e *Engine) Originate(docType, objectID string, controller *Keypair, refs []string, bodyHashH4 []byte) (*Doc, error) {
	def, err := e.Reg.Get(docType)
	if err != nil {
		return nil, err
	}
	d := &Doc{DocType: docType, ObjectID: objectID, State: def.Initial, Refs: refs, BodyHashH4: bodyHashH4, Controller: controller, Seq: 0}
	env, err := e.recAndEnvelope(d, cc.OpInsert, nil)
	if err != nil {
		return nil, err
	}
	tx := transaction.NewTransaction()
	uw, err := e.unlock(e.Wallet)
	if err != nil {
		return nil, err
	}
	if err := tx.AddInputFrom(e.wallet.txid, e.wallet.vout, e.Wallet.LockHex, e.wallet.sats, uw); err != nil {
		return nil, err
	}
	change := e.wallet.sats - stateSats
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: stateSats, LockingScript: env})
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: change, LockingScript: e.Wallet.Lock})
	txid, err := e.send(tx)
	if err != nil {
		return nil, err
	}
	e.wallet = utxo{txid, 1, change}
	d.Txid, d.Vout, d.Sats, d.LockHex = txid, 0, stateSats, hex.EncodeToString(env.Bytes())
	return d, nil
}

// Transition fires a DFA event: validates it, spends the current state UTXO, creates the next state.
// newController (optional) re-keys control — this is B/L endorsement / CUSTODY_TRANSFER (SYS-LOG-006/007).
func (e *Engine) Transition(d *Doc, event string, newController *Keypair) (*Doc, error) {
	def, err := e.Reg.Get(d.DocType)
	if err != nil {
		return nil, err
	}
	to, ok := def.next(d.State, event)
	if !ok {
		return nil, fmt.Errorf("%s: no transition from %q on %q", d.DocType, d.State, event)
	}
	ctrl := d.Controller
	if newController != nil {
		ctrl = newController
	}
	nd := &Doc{DocType: d.DocType, ObjectID: d.ObjectID, State: to, Refs: d.Refs, BodyHashH4: d.BodyHashH4, Controller: ctrl, Seq: d.Seq + 1}
	env, err := e.recAndEnvelope(nd, cc.OpUpdate, mustHex(d.Txid))
	if err != nil {
		return nil, err
	}
	tx := transaction.NewTransaction()
	ud, err := e.unlock(d.Controller) // current controller authorises the transition
	if err != nil {
		return nil, err
	}
	uw, err := e.unlock(e.Wallet)
	if err != nil {
		return nil, err
	}
	if err := tx.AddInputFrom(d.Txid, d.Vout, d.LockHex, d.Sats, ud); err != nil {
		return nil, err
	}
	if err := tx.AddInputFrom(e.wallet.txid, e.wallet.vout, e.Wallet.LockHex, e.wallet.sats, uw); err != nil {
		return nil, err
	}
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: d.Sats, LockingScript: env})
	tx.AddOutput(&transaction.TransactionOutput{Satoshis: e.wallet.sats, LockingScript: e.Wallet.Lock})
	txid, err := e.send(tx)
	if err != nil {
		return nil, err
	}
	e.wallet = utxo{txid, 1, e.wallet.sats}
	nd.Txid, nd.Vout, nd.Sats, nd.LockHex = txid, 0, d.Sats, hex.EncodeToString(env.Bytes())
	_ = def.isFinal
	return nd, nil
}

func (e *Engine) send(tx *transaction.Transaction) (string, error) {
	if err := tx.Sign(); err != nil {
		return "", err
	}
	txid, err := e.C.SendRawTransaction(tx.Hex())
	if err != nil {
		return "", err
	}
	return txid, e.mine()
}

// History is the cold-rebuilt, tag-verified state sequence of a document lineage.
type History struct {
	States   []string
	Final    DecodedState
	Verified int
}

// Verify walks the document lineage from headTxid via prev_txid, tag-verifies each transition, and
// returns the ordered state history (SYS-EDI-003 discoverability + integrity of the lifecycle).
func (e *Engine) Verify(headTxid, objectID string) (History, error) {
	var h History
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
			return h, err
		}
		tx, err := transaction.NewTransactionFromHex(rawHex)
		if err != nil {
			return h, err
		}
		var data []byte
		for _, o := range tx.Outputs {
			if o.LockingScript == nil {
				continue
			}
			if d, err := bsvscript.ExtractEnvelopeData(o.LockingScript); err == nil {
				if rec, derr := cc.DecodeRecord(d); derr == nil && rec.Message.ColumnID == "state" && string(rec.Message.RowID) == objectID {
					data = d
					break
				}
			}
		}
		if data == nil {
			return h, fmt.Errorf("no state record for %s on %s", objectID, txid)
		}
		rec, err := cc.DecodeRecord(data)
		if err != nil {
			return h, err
		}
		rev = append(rev, ent{rec.Message, rec.ChangeImage, rec.Tag})
		if len(rec.Message.PrevTxid) == 0 {
			txid = ""
		} else {
			txid = hex.EncodeToString(rec.Message.PrevTxid)
		}
	}
	for i := len(rev) - 1; i >= 0; i-- {
		en := rev[i]
		_, gv, err := cc.GeneratorValue(en.m)
		if err != nil {
			return h, err
		}
		cs, err := cc.CommonSecretAsWriter(e.WriterPriv, e.CpPub, gv)
		if err != nil {
			return h, err
		}
		kh, err := cc.DeriveHMACKey(cs, en.m)
		if err != nil {
			return h, err
		}
		if hex.EncodeToString(cc.Tag(kh, en.img)) != hex.EncodeToString(en.tag) {
			return h, fmt.Errorf("tag verify failed at seq %d", en.m.Seq)
		}
		ds, err := decodePayload(en.img)
		if err != nil {
			return h, err
		}
		h.States = append(h.States, ds.State)
		h.Final = ds
	}
	h.Verified = len(rev)
	return h, nil
}

func mustHex(s string) []byte { b, _ := hex.DecodeString(s); return b }
