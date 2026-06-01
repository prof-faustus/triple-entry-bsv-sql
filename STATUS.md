# Status — SQL (Triple-Entry BSV SQL)

_Last updated: 2026-06-01_

**Overall:** Active/in-progress

## What this is
A monorepo for a tokenised commercial system on Bitcoin SV implementing triple-entry accounting: a fork of PostgreSQL 18 with native BSV embedding, so ordinary SQL writes are mirrored per-field onto BSV as an immutable, ECDH-HMAC-bound, hash-chained "third entry". Four packages share the substrate: triple-entry ledger, tokenised EDI documents, multi-currency tokenised cash, and tokenised logistics/goods.

## Current state
- Substantial multi-language codebase present: `crypto-core/` (C + TS/Go parity), `pg-fork/`, `tokenisation/`, `edi-dfa/`, `edi-bridge/`, `doc-render/`, `logistics/`, `sdk-ts/`, `services-go/`, plus `spec/` and `tests/`.
- `spec/STATUS.md` (the project's own per-requirement tracker) reports Phases 0–7 implemented as e2e-verified vertical slices on PostgreSQL 18.4 + Teranode regtest (WSL), and additionally re-run against a funded SV Node `bitcoind` regtest wallet; SPV/BURI proof layer validated read-only against the public BSV testnet via WhatsOnChain.
- `evidence/` holds 10+ captured run-output files (full-stack, keystone, SPV testnet, hardening, token, EDI/logistics, PG triple-entry, PG18 install, crypto parity, on-chain confirmations).
- Note: these completion/test claims are from the repo's own docs and captured evidence logs; they were not independently re-run this pass.
- Known remaining work per the repo's own notes: broadcasting own third-entry txs on testnet/mainnet remains fund-gated; production hardening of the documented slices ongoing.

## Version control
- Git: yes, branch master, last commit `abb4eb6 Production-hardening: QR in PDF, confidential-commitment path, SQL te_render_pdf`, working tree dirty (1 file: `services-go/cmd/tewriter/main.go` modified).

## How to verify / build
- No top-level Makefile/package.json; build is per-component (C toolchain for `pg-fork`/`crypto-core`, Go for `services-go`, TS for `sdk-ts`). Declared runner scripts referenced in `spec/STATUS.md`: `services-go/run-all-svnode-wsl.sh`, `run-spvtestnet-wsl.sh`, `run-harden-e2e-wsl.sh`, `node-docker/lib/reset-regtest.sh` (require WSL + Docker + a BSV/Teranode or SV Node regtest node).
- Not assessed this pass (no builds/tests run).
