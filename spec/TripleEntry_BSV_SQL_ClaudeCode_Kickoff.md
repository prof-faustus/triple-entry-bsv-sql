# Claude Code Kickoff — Tokenised Commercial System on BSV (Triple-Entry SQL)

Paste the block below into Claude Code at the repo root. It drives the phased build defined by `TripleEntry_BSV_SQL_Build_Spec_v8.md` (the "Spec"). It is written to run autonomously through the phases that need no external credentials or human decisions, and to **stop and ask** at the points that genuinely do.

---

```
You are building the system specified in TripleEntry_BSV_SQL_Build_Spec_v8.md (the "Spec"), in this repository. Treat the Spec as the contract. Read it in full before writing code, and re-read the relevant section before each task. The Spec's Appendix A is the requirements index; Appendix B is the definition of done.

NON-NEGOTIABLE RULES (a violation of any is a build failure, not a style issue):
- BSV ONLY. No BTC constructs: no SegWit, Taproot, witness, Lightning, RBF. Confusing BTC and BSV fails the build.
- NO P2SH. P2SH is consensus-prohibited on BSV. Use native locking scripts only: P2PKH, bare OP_CHECKSIG, bare OP_CHECKMULTISIG.
- NO OP_RETURN — at all. Carry ALL on-chain data (state, token metadata, ECDH-HMAC tags, prev-references) as pushdata inside SPENDABLE locking scripts: an unexecuted OP_FALSE OP_IF <data> OP_ENDIF branch, or <data> OP_DROP before the authorisation opcodes. Every data output stays spendable.
- The NODE is Teranode (teranode-quickstart Docker; regtest for dev/test). It is a Go microservices cluster, not a single bitcoind. Do NOT assume its RPC method names, service endpoints, or event topics — read them from the current Teranode docs and pin them in VERIFY-LOG.md before integrating.
- The DATABASE is a fork of PostgreSQL 18 (PostgreSQL License). Preserve upstream copyright/licence notices.
- The KEYSTONE is the ECDH-HMAC hash-chain TX log (Spec Section 5): per SQL change c, M(c) -> GV=SHA256(M) -> deterministic sub-keys -> common secret CS (ECDH) -> K_hmac=HKDF(CS) -> tag=HMAC(K_hmac, change_image); tag in a spendable locking script; each tx carries prev_txid; the DB is rebuildable from the chain alone.
- SQL-SIMPLE: the user runs ordinary SQL; all on-chain mechanics are invisible and built into the fork.
- HONESTY: never fabricate a value, an interface, or a patent detail. If a fact must be confirmed against a current source, mark it [VERIFY] and resolve it before code depends on it. State assumptions explicitly; do not invent.

WORKING DISCIPLINE:
- Build strictly in the Spec's phase order (Section 13). Do not start a phase until the prior phase's exit criteria pass.
- Keep a VERIFY-LOG.md (resolve every [VERIFY], incl. Teranode interface names and in-force miner policy) and a DECISIONS.md (the SYS-DECIDE-* values; the locked ones are in Spec Section 14).
- Cross-implementation crypto vectors (C fork vs TS/Go) must match exactly; treat divergence as a release blocker.
- After each phase, run the relevant Appendix B checks and report pass/fail per requirement ID before proceeding.

PHASES — run autonomously through Phase 3, then checkpoint:

PHASE 0 — Spec freeze. Create the repo skeleton from Spec Section 13. Read Teranode docs; pin the teranode-quickstart regtest setup, RPC/service/event interfaces, and image tags in VERIFY-LOG.md. Pin PostgreSQL 18 minor. Confirm regtest comes up and produces blocks on demand.

PHASE 1 — Crypto core + KAT. Implement ECDH common-secret (EP3860037A1 method), HKDF, HMAC-SHA256, commitments, and the canonical length-prefixed encoder/decoder, in the shared core, with C and TS/Go bindings. Exit: known-answer vectors green and identical across implementations (Appendix B.1).

PHASE 2 — Node + hash-chain log. Stand up the Teranode regtest node in Docker. Build, sign (SIGHASH_ALL|FORKID), and broadcast a hash-chained ECDH-HMAC TX stream where data rides in spendable scripts (no OP_RETURN, no P2SH). Implement tag discovery and cold-rebuild of a toy stream. Exit: Appendix B.2, B.5, B.6 on a toy stream.

PHASE 3 — PostgreSQL fork. Fork PostgreSQL 18; implement committed-write interception (decide WAL/extension/in-core per SYS-DECIDE-002), the transactional outbox, async + sync journalling modes, te_verify(), and te_render_pdf(). Exit: ordinary SQL on a real schema yields a correct, verifiable on-chain triple-entry log; full cold-rebuild asserts byte-equality (Appendix B.2–B.6).

>>> CHECKPOINT: stop here and report. Summarise Phase 0–3 results per requirement ID. Do not begin Phase 4 until I confirm. <<<

PHASE 4 — Definable token. Implement the token-definition schema (SYS-TOK-005): any item/money unit, issuer-defined. Implement cash profiles (issuer-backed, satoshi-tagged, pegged), goods tokens, atomic two-token swaps, and the external-linkage ADAPTER CONTRACT (SYS-TOK-006) for CBDC/stablecoin/other-coin links — implement the on-chain side and the adapter interface only; do NOT integrate any real external rail without its actual interface and my go-ahead. Exit: Appendix B.7.

PHASE 5 — EDI DFA + bridge + logistics. Implement every document DFA (SYS-EDI-002), the consignment lifecycle, multi-party custody, bill-of-lading-as-token, ownership (US11210372) and integrity (GB2558485A) checks, delivery-versus-payment, and the optional, per-partner X12/EDIFACT bridge (SYS-EDI-005/006). Exit: Appendix B.8, B.9.

PHASE 6 — Proofs / custody / overlay / computation. SPV inclusion proofs + BURI export; threshold custody (US11671255 signing without key reconstruction; EP3259724B1 share storage); overlay addressing (EP4046048B1 CKD hierarchy); staked-computation resolution feeding DFA events (US20240364498A1). Exit: Appendix B.10.

PHASE 7 — Hardening. Reorg handling, outbox idempotency across restarts, confirmation-depth gating, security review. Move from regtest to teratestnet only after Appendix B passes end-to-end. Exit: Appendix B.11, B.12, full Appendix A satisfied or explicitly waived.

STOP-AND-ASK (do not proceed without me): mainnet deployment; wiring any real CBDC/stablecoin/external rail; any step needing live credentials, funds, or a miner-policy survey; any situation where the only way forward would require an assumption you cannot verify.

Begin with Phase 0 now.
```

---

## Notes for the operator

- The kickoff stops after Phase 3 (the working triple-entry core: SQL in, verifiable on-chain hash chain out, cold-rebuildable) so you can review before the larger token/EDI/logistics build.
- The two locked decisions already in the Spec: Teranode node (`SYS-DECIDE-010`), X12/EDIFACT bridge provided (`SYS-DECIDE-005`), definable token + CBDC linkage (`SYS-DECIDE-001`), PostgreSQL 18 (`SYS-DECIDE-008`). The still-open implementation decisions (`SYS-DECIDE-002/003/004/006/007/009`) are flagged in the Spec for you to set in DECISIONS.md.
- The external-rail adapters (CBDC, stablecoin, other token protocols) are deliberately built as contracts only; integrating a real rail needs that rail's actual interface and is gated behind STOP-AND-ASK.
