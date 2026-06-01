# Teratestnet path (Phase 7 → public test network) — GATED

The Spec moves from regtest to **teratestnet** only after Appendix B passes end-to-end (done on
regtest) and is an explicit **STOP-AND-ASK** step: it touches a public network, needs **funded
teratestnet keys**, and requires a **miner-policy survey** (`SYS-VERIFY-LIST`, `SYS-DECIDE-004`). This
file prepares the path; it does **not** transact on a public network autonomously.

## What is prepared (config only)
`teratestnet-up.sh` writes the correct per-network knobs into the quickstart `.env`:
- `network=teratestnet`; `minminingtxfee=0.00000001` (1 sat/kb); `blockmaxsize`/`excessiveblocksize`
  capped at **1 GiB** (teratestnet caps); `COMPOSE_PROFILES=legacy,p2p`; an explicit
  `legacy_config_ConnectPeers` peer (teratestnet has **no DNS seeder**).
- Pins the Chronicle-current Teranode image (`TERANODE_VERSION`, `VERIFY-LOG.md` B8/B9/B10).
The script **refuses to start** unless `TERATESTNET_CONFIRM=1` is set.

## What is required before going live (operator must provide / confirm)
1. **Go-ahead** to operate on the public teratestnet (this is the STOP-AND-ASK).
2. A **funded teratestnet key** (coins from the teratestnet faucet) for fees/coinbase-equivalent —
   the regtest `generatetoaddress` self-funding path does **not** exist on a shared network.
3. A current **miner-policy survey** (in-force `maxscriptsizepolicy`, fees, dust) per `VERIFY-LOG.md`
   D1–D4, and empirical confirmation of Chronicle sighash acceptance (`VERIFY-LOG.md` B10).
4. A reachable **peer/endpoint** for `legacy_config_ConnectPeers` (and, for full mode, public
   asset/P2P endpoints — quickstart does not provision TLS/reverse-proxy).

## How to proceed once the above are in hand
```
# 1) configure
wsl -u root -- bash /mnt/d/claude/SQL/node-docker/teratestnet-up.sh
# 2) bring up + sync (only after providing funds + confirming)
wsl -u root -- bash -lc 'cd /root/teranode-quickstart && TERATESTNET_CONFIRM=1 ./start.sh'
# 3) point the writers/e2e at the synced node; fund from the faucet key instead of generatetoaddress
```
The application layer (crypto core, envelope, keystone, token/EDI/logistics engines, writer) is
network-agnostic — only the **funding source** changes (faucet-funded UTXO instead of a regtest
coinbase). No code change is expected; the gate is operational (funds + go-ahead + policy survey).
