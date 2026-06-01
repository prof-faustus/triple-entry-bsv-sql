// Package bsvscript builds native-BSV spendable locking scripts that carry data,
// per SYS-ENC-001 / SYS-CON-002 / SYS-CON-008: data rides as pushdata inside a
// SPENDABLE locking script (never OP_RETURN), and authorisation is native P2PKH
// (no P2SH wrapper). Two carriers are provided, both keeping the output spendable:
//
//	(a) OP_FALSE OP_IF <data> OP_ENDIF  <P2PKH...>      — unexecuted data branch
//	(b) <data> OP_DROP  <P2PKH...>                       — pushed then dropped
//
// The trailing P2PKH (OP_DUP OP_HASH160 <pkh> OP_EQUALVERIFY OP_CHECKSIG) makes the
// output a normal spendable UTXO node of the lineage (SYS-ENC-002).
package bsvscript

import (
	"errors"

	"github.com/bsv-blockchain/go-sdk/script"
)

// BuildEnvelopeIf returns carrier (a): OP_FALSE OP_IF <data> OP_ENDIF + P2PKH(pkh).
func BuildEnvelopeIf(data []byte, pkh []byte) (*script.Script, error) {
	if len(pkh) != 20 {
		return nil, errors.New("pkh must be 20 bytes (hash160)")
	}
	s := &script.Script{}
	if err := s.AppendOpcodes(script.OpFALSE, script.OpIF); err != nil {
		return nil, err
	}
	if err := s.AppendPushData(data); err != nil {
		return nil, err
	}
	if err := s.AppendOpcodes(script.OpENDIF, script.OpDUP, script.OpHASH160); err != nil {
		return nil, err
	}
	if err := s.AppendPushData(pkh); err != nil {
		return nil, err
	}
	if err := s.AppendOpcodes(script.OpEQUALVERIFY, script.OpCHECKSIG); err != nil {
		return nil, err
	}
	return s, nil
}

// BuildEnvelopeDrop returns carrier (b): <data> OP_DROP + P2PKH(pkh).
func BuildEnvelopeDrop(data []byte, pkh []byte) (*script.Script, error) {
	if len(pkh) != 20 {
		return nil, errors.New("pkh must be 20 bytes (hash160)")
	}
	s := &script.Script{}
	if err := s.AppendPushData(data); err != nil {
		return nil, err
	}
	if err := s.AppendOpcodes(script.OpDROP, script.OpDUP, script.OpHASH160); err != nil {
		return nil, err
	}
	if err := s.AppendPushData(pkh); err != nil {
		return nil, err
	}
	if err := s.AppendOpcodes(script.OpEQUALVERIFY, script.OpCHECKSIG); err != nil {
		return nil, err
	}
	return s, nil
}

// ExtractEnvelopeData recovers the data payload from either carrier form.
func ExtractEnvelopeData(s *script.Script) ([]byte, error) {
	ops, err := s.ParseOps()
	if err != nil {
		return nil, err
	}
	if len(ops) < 2 {
		return nil, errors.New("script too short for an envelope")
	}
	// (a) OP_FALSE OP_IF <data> OP_ENDIF ...
	if ops[0].Op == script.OpFALSE && ops[1].Op == script.OpIF {
		if len(ops) < 4 || ops[3].Op != script.OpENDIF {
			return nil, errors.New("malformed OP_IF envelope")
		}
		if len(ops[2].Data) == 0 {
			return nil, errors.New("empty envelope payload")
		}
		return ops[2].Data, nil
	}
	// (b) <data> OP_DROP ...
	if len(ops[0].Data) > 0 && ops[1].Op == script.OpDROP {
		return ops[0].Data, nil
	}
	return nil, errors.New("no recognised data envelope prefix")
}

// EnvelopePubKeyHash returns the 20-byte pubkey hash from the native P2PKH spend tail
// (OP_DUP OP_HASH160 <pkh> OP_EQUALVERIFY OP_CHECKSIG), regardless of the data prefix.
func EnvelopePubKeyHash(s *script.Script) ([]byte, error) {
	ops, err := s.ParseOps()
	if err != nil {
		return nil, err
	}
	n := len(ops)
	// tail: OP_DUP OP_HASH160 <pkh(20)> OP_EQUALVERIFY OP_CHECKSIG
	if n < 5 || ops[n-1].Op != script.OpCHECKSIG || ops[n-2].Op != script.OpEQUALVERIFY ||
		ops[n-4].Op != script.OpHASH160 || ops[n-5].Op != script.OpDUP {
		return nil, errors.New("no native P2PKH spend tail")
	}
	pkh := ops[n-3].Data
	if len(pkh) != 20 {
		return nil, errors.New("spend-tail pubkey hash is not 20 bytes")
	}
	return pkh, nil
}

// AssertNativeSpendable enforces the build-failure rules over a produced locking script:
// no P2SH pattern, no OP_RETURN anywhere, and a native P2PKH spend tail so the output
// is genuinely spendable (SYS-CON-002, SYS-CON-008, SYS-ENC-001/002; Appendix B.5).
func AssertNativeSpendable(s *script.Script) error {
	if s.IsP2SH() {
		return errors.New("forbidden: P2SH pattern (SYS-CON-002)")
	}
	ops, err := s.ParseOps()
	if err != nil {
		return err
	}
	for _, op := range ops {
		if op.Op == script.OpRETURN {
			return errors.New("forbidden: OP_RETURN output (SYS-CON-008)")
		}
	}
	// spend tail must be ... OP_EQUALVERIFY OP_CHECKSIG (native P2PKH)
	n := len(ops)
	if n < 5 || ops[n-1].Op != script.OpCHECKSIG || ops[n-2].Op != script.OpEQUALVERIFY {
		return errors.New("not spendable: missing native P2PKH authorisation tail (SYS-ENC-002)")
	}
	return nil
}
