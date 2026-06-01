# Evidence — captured run outputs (2026-06-01)

All phases were built and verified end-to-end on **PostgreSQL 18.4 + a running funded SV Node**
(regtest, RPC :18443, ~3,500 BSV) in WSL, with the SPV/BURI proof layer additionally verified against
**live BSV testnet** (WhatsOnChain). Every transaction below was **accepted by the SV Node under full
BSV consensus**; on-chain presence is shown in `10_onchain-confirmations.txt`.

| File | What it proves |
|---|---|
| `01_full-stack-svnode.txt` | Full stack funded from the SV Node wallet + accepted: P4 token (mint/transfer/redeem/swap), P5 EDI+logistics (22 DFAs, custody, ownership/integrity, DvP, B/L-as-token), P3 PG triple-entry (SQL→third entries→cold-rebuild == live DB), P7 hardening (confirmation gating, reorg with reconsiderblock restore, idempotency). All `rc=0`. |
| `02_keystone-svnode.txt` | Hash-chained ECDH-HMAC third-entry stream funded from the wallet, accepted by the SV Node; cold-rebuild == source. |
| `03_spv-live-bsv-testnet.txt` | SPV/BURI (`SYS-PROOF-*`) verified against a **real BSV testnet block** (height 1,738,646, 130 txs): Merkle root reproduced == live header; real tx inclusion + BURI verified. No funds. |
| `04_hardening.txt` | `SYS-PG-007`: confirmation-depth gating, reorg re-evaluation (+ clean `reconsiderblock` restore), outbox idempotency. |
| `05_token.txt` | `SYS-TOK-*`, `SYS-CASH-*`: definable token (data-defined, no code change), 3 cash profiles + goods + CBDC adapter-contract, atomic swap. |
| `06_edi-logistics.txt` | `SYS-EDI-*`, `SYS-LOG-*`: 22 document DFAs, cross-refs, consignment/custody, ownership (US11210372) + integrity (GB2558485A), DvP, B/L title transfer. |
| `07_pg-triple-entry.txt` | `SYS-PG-*`: ordinary SQL → atomic trigger capture → on-chain third entries → cold-rebuild == live DB. |
| `08_pg18-install.txt` | PostgreSQL 18.4 fork (PGDG) install + `te` schema bring-up. |
| `09_crypto-parity.txt` | `SYS-TEST-003` / Appendix B.1: crypto core **C == TS == Go byte-for-byte** (14 TS, Go, 40/40 C; incl. RFC-4231/5869 + NIST KATs). |
| `10_onchain-confirmations.txt` | The Phase-4 token mint and Phase-3 third-entry txs **confirmed on the running funded node**. |

Reproduce: `node-docker/regtest-up.sh` (Teranode), `pg-fork/install-pg18.sh` (PG18), and the
`services-go/run-*-wsl.sh` / `pg-fork/run-pg-e2e-wsl.sh` runners. Spec/decisions/verify/security:
`spec/`.
