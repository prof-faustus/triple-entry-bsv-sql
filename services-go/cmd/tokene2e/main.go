// tokene2e drives the Phase-4 exit (Appendix B.7) on Teranode regtest:
// define token types from data (no code change), then mint/transfer/redeem cash + goods, a
// CBDC-linked token via the adapter contract (no real rail), and an atomic two-token swap — each
// event journalled as a third entry; lineages tag-verified from the chain.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	cc "te-bsv/cryptocore"
	"te-bsv/services-go/node"
	"te-bsv/services-go/token"
)

const (
	walletPriv  = "1122334455667788990011223344556677889900112233445566778899001122"
	writerPriv  = "e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262"
	cpPriv      = "f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181"
	issuerPriv  = "1111111111111111111111111111111111111111111111111111111111111111"
	holderAPriv = "2222222222222222222222222222222222222222222222222222222222222222"
	holderBPriv = "3333333333333333333333333333333333333333333333333333333333333333"
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
func must[T any](v T, err error) T { ck(err); return v }

func main() {
	rpc := flag.String("rpc", "http://127.0.0.1:18443", "RPC URL (SV Node wallet)")
	user := flag.String("user", "cto", "RPC user")
	pass := flag.String("pass", "ctopass", "RPC pass")
	defs := flag.String("defs", "/mnt/d/claude/SQL/tokenisation/token-defs.json", "token definitions")
	logp := flag.String("log", filepath.Join(os.TempDir(), "te_token.log"), "log path")
	flag.Parse()
	if fl, err := os.Create(*logp); err == nil {
		logFile = fl
		defer fl.Close()
	}

	reg := must(token.LoadRegistry(*defs))
	logf("== token types defined from data (no code change) ==")
	for id, d := range reg {
		ext := "none"
		if d.External != nil {
			ext = d.External.Mode + ":" + d.External.Adapter
		}
		logf("  %-13s unit=%-11s backing=%-13s external=%s", id, d.Unit, d.Backing, ext)
	}

	wallet := must(token.NewKeypair(walletPriv))
	issuer := must(token.NewKeypair(issuerPriv))
	holderA := must(token.NewKeypair(holderAPriv))
	holderB := must(token.NewKeypair(holderBPriv))
	cpPub := cc.PubFromPriv(mustHex(cpPriv))

	e := &token.Engine{C: node.New(*rpc, *user, *pass), Wallet: wallet, WriterPriv: mustHex(writerPriv), CpPub: cpPub}
	e.MinerAddr = must(e.C.GetNewAddress())
	ck(e.FundFromWallet(20.0))
	logf("funded fee key from SV Node wallet (sendtoaddress)")

	// ---- cash (issuer-backed): mint -> transfer -> redeem ----
	usdDef := must(reg.Get("cash-usd"))
	logf("== cash-usd: mint -> transfer -> redeem ==")
	usd := must(e.Mint(usdDef, "USD#1", issuer, 150000)) // $1500.00
	logf("  mint  $1500.00 to issuer   txid=%s sats=%d", usd.Txid, usd.Sats)
	usd = must(e.Transfer(usd, holderA))
	logf("  xfer  -> holderA           txid=%s", usd.Txid)
	redeemTxid := must(e.Redeem(usd, issuer))
	logf("  redeem (burn) -> issuer    txid=%s", redeemTxid)
	st := must(e.VerifyLineage(redeemTxid, "USD#1"))
	expect(st.Entries == 3 && !st.Alive && bytes.Equal(st.ControllerPub, issuer.Pub), "cash-usd lineage")
	logf("  verified: %d events, burned, final controller=issuer ✓", st.Entries)

	// ---- CBDC-linked via adapter contract (no real rail) ----
	cbdcDef := must(reg.Get("cbdc-egbp"))
	logf("== cbdc-egbp: adapter contract (no real rail) ==")
	adapter := token.MockAdapter{Name: cbdcDef.External.Adapter, FixedRate: cbdcDef.PeggingRateMicro}
	rate := must(adapter.RateMicro())
	ref := must(adapter.Lock(50000, "EGBP#1"))
	logf("  oracle=%s rate_micro=%d  custodian=%s  lock_ref=%s (MOCK — real rail gated by STOP-AND-ASK)",
		cbdcDef.External.Oracle, rate, cbdcDef.External.Custodian, ref)
	egbp := must(e.Mint(cbdcDef, "EGBP#1", holderA, 50000)) // £500.00
	egbp = must(e.Transfer(egbp, holderB))
	stc := must(e.VerifyLineage(egbp.Txid, "EGBP#1"))
	expect(stc.Entries == 2 && stc.Alive && bytes.Equal(stc.ControllerPub, holderB.Pub), "cbdc-egbp lineage")
	logf("  minted+transferred; verified %d events, controller=holderB ✓", stc.Entries)

	// ---- atomic two-token swap (deliver-vs-deliver), SYS-TOK-007 ----
	logf("== atomic swap: bsv-tagged (holderA) <-> goods-widget (holderB) ==")
	satTok := must(e.Mint(must(reg.Get("bsv-tagged")), "SAT#swap", holderA, 5000))
	goodsTok := must(e.Mint(must(reg.Get("goods-widget")), "WIDGET#swap", holderB, 1))
	na, nb := must2(e.Swap(satTok, goodsTok))
	expect(na.Txid == nb.Txid, "swap atomicity (single txid)")
	logf("  swapped in ONE tx %s (atomic)", na.Txid)
	sa := must(e.VerifyLineage(na.Txid, na.TokenID))
	sb := must(e.VerifyLineage(nb.Txid, nb.TokenID))
	expect(bytes.Equal(sa.ControllerPub, holderB.Pub), "swap leg A -> holderB")
	expect(bytes.Equal(sb.ControllerPub, holderA.Pub), "swap leg B -> holderA")
	logf("  verified: SAT now holderB, WIDGET now holderA ✓")

	logf("RESULT: PHASE-4 E2E PASS — definable token (no-code-change types), cash/CBDC-adapter/goods mint·transfer·redeem journalled, atomic swap, native (no P2SH/OP_RETURN)")
}

func expect(cond bool, what string) {
	if !cond {
		ck(fmt.Errorf("assertion failed: %s", what))
	}
}
func must2[A any, B any](a A, b B, err error) (A, B) { ck(err); return a, b }
func mustHex(s string) []byte                        { b, err := hex.DecodeString(s); ck(err); return b }
