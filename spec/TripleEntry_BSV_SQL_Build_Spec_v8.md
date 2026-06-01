# Tokenised Commercial System on Bitcoin SV
## Integrated Build Specification — Tokenised EDI, Multi-Currency Cash, Tokenised Logistics, and a Forked-PostgreSQL Triple-Entry Ledger
### Version 8.0 (Release) — Claude Code Implementation Brief

---

## 0. Document Control and Integrity Rules

| Field | Value |
|---|---|
| Title | Tokenised Commercial System on BSV — Integrated Build Specification |
| Version | 8.0 (build-ready release) |
| Target chain | Bitcoin SV (BSV) **only**. No BTC constructs. No P2SH (disabled at Genesis). No OP_RETURN at all. |
| Test substrate | A full BSV node running in Docker (regtest/testnet). |
| Substrate | Builds on the CTO confidential-object primitive (`CTO_BSV_Build_Spec_v1.md`). |
| Packages | (A) Forked-PostgreSQL triple-entry ledger; (B) Tokenised EDI / commercial documents; (C) Multi-currency tokenised cash; (D) Tokenised logistics / goods. |
| Implementation host | Claude Code |
| Languages | C (PostgreSQL fork + native BSV), Go (services/node glue), TypeScript (SDK/clients), BSV Script (on-chain). |

### 0.1 Integrity rules governing this document (non-negotiable)

`SYS-INTEG-001` No requirement, claim, or design element in this document may be grounded in the **abstract** of a patent. Anything attributed to a patent is grounded in that patent's **claims and description, read in full**. Sections that depend on a patent not yet read in full are marked `[PENDING READ: <patent>]` and assert nothing about that patent's content beyond its identity and subject.

`SYS-INTEG-002` Where a value, opcode behaviour, library API, node RPC, or ecosystem fact must be confirmed against a current authoritative source, the text carries `[VERIFY]`. Each `[VERIFY]` is a gate resolved before code relies on it, recorded in `VERIFY-LOG.md`.

`SYS-INTEG-003` No deployment decision is made silently. Open decisions are recorded in Section 14 (`SYS-DECIDE-*`) and chosen by the operator, not assumed.

`SYS-INTEG-004` This system implements methods taught in nChain patents. Freedom-to-operate and licensing are legal questions outside this document's scope.

### 0.2 Grounding provenance (read state of each source)

**Read in full (claims + description) — used as authoritative here:**
- **EP3748903A1** — Universal Tokenisation System (Wright, Savanah). Grounds Section 8.
- **US20220253835A1** — Determining the State of a Machine-Executable Contract via a Blockchain (Jimenez-Delgado, Wright). Grounds Sections 6 and 9.
- **EP3860037A1** — Cryptographic Method and System for Secure Extraction of Data from a Blockchain (Wright). Grounds Section 5 (the ECDH-HMAC keystone) and Section 6.3.
- **US11210372** — Verifying Ownership of a Digital Asset using a Distributed Hash Table and a P2P Distributed Ledger (Savanah, Wright). Grounds Section 10 (goods records + ownership).
- **GB2558485A** — Verifying **Integrity** of a Digital Asset using a Distributed Hash Table and a P2P Distributed Ledger (Savanah, Wright). The copy supplied is a 1-page cover only; the **full claims and description were read from the same patent family** — granted US10579779B2 / published WO2017195160A1, identical disclosure and priority (PCT/IB2017/052800, priority 13 May 2016). Grounds Section 10.6 (integrity verification).
- **WO2022100946A1** — Providing proof that a data item of a transaction exists on a blockchain (Zhang, Ammar, Davies, Wright). Grounds Section 11.1.
- **WO2022214264A1** — Verifying that an identified transaction is stored in a blockchain (BURI) (Wright, Graham, Davies). Grounds Section 11.1.
- **WO2025119666A1** — Enabling verification of presence of a data item on a blockchain (Wright, Lunardon). Grounds Section 11.1.
- **US11671255** — Threshold Digital Signature Method and System (Savanah, Wright). Grounds Section 11.2.
- **EP3259724B1** — Secure multiparty loss-resistant storage and transfer of cryptographic keys (nChain). Grounds Section 11.2.
- **EP4046048B1** — Mapping Keys to a Blockchain Overlay Network (nChain). Grounds Section 11.3.
- **US20240364498A1** — Blockchain for General Computation (Trevethan, Wright). Grounds Section 11.4.

**Academic grounding for the triple-entry model (read in full):**
- **Ijiri (1986)** — Framework for triple-entry bookkeeping. Grounds Section 7.
- **McCarthy (1982)** — REA accounting model. Grounds Section 7.3.
- **Cai (2019)** — Triple-entry accounting with blockchain. Grounds Section 7.4.

**Source note:** every patent in the corpus is now read in full. GB2558485A was supplied as a cover sheet only; its full disclosure was read from the granted family member US10579779B2 (= WO2017195160A1), which is the same invention. No source in this spec is grounded on an abstract.

---

## 1. Scope

`SYS-SCOPE-001` The system **MUST** deliver a single integrated platform with four packages sharing one substrate:

- **Package A — Triple-entry ledger.** A fork of PostgreSQL with native BSV embedded, so that ordinary SQL writes are mirrored, per-field and per-change, onto BSV as an immutable, ECDH-HMAC-bound, hash-chained third entry. The user experience is plain SQL.
- **Package B — Tokenised EDI.** Commercial documents (purchase order, invoice, payment note, shipping/logistics documents) modelled as deterministic-finite-automaton lifecycles whose states are UTXOs.
- **Package C — Tokenised cash / value.** A definable exchangeable token (`SYS-TOK-005`) for any money unit — any fiat currency, CBDC, or token-based coin/stablecoin on BSV or beyond — with profiles selectable and external linkage per `SYS-DECIDE-001`/`SYS-TOK-006`.
- **Package D — Tokenised logistics / goods.** Goods represented as tokens whose detailed records live off-chain (forked Postgres and/or DHT), with on-chain control and transfer.

`SYS-SCOPE-002` The packages **MUST** interoperate: an EDI invoice (B) references a cash settlement (C) and a goods consignment (D), and every state change in all packages is journalled by the triple-entry ledger (A). All four ride the CTO confidential substrate and the BSV node (Section 3).

---

## 2. Non-Negotiable Constraints

`SYS-CON-001` **BSV only.** No SegWit, Taproot, Tapscript, witness, Lightning, RBF, or any BTC construct. Confusing BTC and BSV is a build failure.

`SYS-CON-002` **No P2SH.** Pay-to-Script-Hash is disabled on BSV. All locking scripts are **native**: P2PKH, bare/raw `OP_CHECKSIG` and `OP_CHECKMULTISIG`. The patents that teach P2SH redeem scripts (EP3748903A1, US20220253835A1) are used for their **method**, re-expressed in native BSV; the system is **not** bound to patent script-encoding exactness. P2SH prohibition is a **consensus rule** on BSV; script/pushdata limits are resolved in Section 14.1.

`SYS-CON-008` **No OP_RETURN — at all.** No data may be carried in an `OP_RETURN` output anywhere in the system. All on-chain data (state, token metadata, ECDH-HMAC tags, prev-references) **MUST** be carried as pushdata inside **spendable** locking scripts (Section 4), so every data-bearing output remains a spendable node of the UTXO/overlay lineage. Any use of `OP_RETURN` is a build failure.

`SYS-CON-003` **Every SQL field and every change MUST be bound to an ECDH-keyed HMAC carried in the transaction script and on chain** (Section 5). No accounting-relevant write may exist off-chain without its on-chain ECDH-HMAC node.

`SYS-CON-004` **The on-chain record MUST form a hash chain** (a TX log) such that every transaction is discoverable and the full ledger history is reconstructable by traversal (Section 5.4). 

`SYS-CON-005` **The user-facing interface MUST be SQL.** All on-chain mechanics (key derivation, HMAC tagging, TX assembly, broadcast, hash-chain maintenance, reorg handling) are built into the forked database and are invisible to the SQL user (Section 6).

`SYS-CON-006` **A full BSV node MUST run in Docker** for development and testing; the system MUST operate end-to-end against it on regtest before any testnet/mainnet use (Section 3.3, Section 12).

`SYS-CON-007` The cryptographic honesty boundary of the CTO substrate (forward access revocation provable; erasure of observed plaintext not provable) carries over unchanged. No package may claim otherwise.

---

## 3. Substrate and Node

### 3.1 CTO confidential substrate

`SYS-SUB-001` Confidential payloads (private ledger fields, document contents, goods records) and threshold custody **MUST** use the CTO primitive: secp256k1 ECDH, HKDF, AEAD, SHA-256 commitments, UTXO-lineage objects, time-locked recovery. This spec adds the SQL binding, the ECDH-HMAC hash chain, tokenisation, EDI, and logistics on top.

### 3.2 Why this composes

`SYS-SUB-002` The CTO model already represents an object as a single-threaded UTXO lineage with on-chain control and off-chain confidential payload. The triple-entry third entry is a UTXO lineage whose payload is the ECDH-HMAC of a SQL change; an EDI document is a UTXO lineage whose states are DFA states; a cash or goods token is a UTXO lineage carrying token metadata. One substrate, four payload shapes.

### 3.3 Full BSV node in Docker

`SYS-NODE-001` The system **MUST** ship a Docker composition that runs a full BSV node plus the forked PostgreSQL and the service layer. **Decision (`SYS-DECIDE-010`, locked):** the node is **Teranode** — the next-generation BSV node (Go, microservices, horizontally scalable). It ships via `teranode-quickstart` Docker supporting **mainnet/testnet/teratestnet/regtest**; development and the full test suite run on **regtest**, scaling to teratestnet/mainnet unchanged. SV Node (`bitcoinsv/bitcoin-sv`, C++ monolithic) is **not** the chosen node and is referenced only as the alternative implementation. The **Chronicle** upgrade is mandatory (mainnet 7 April 2026), so the Teranode build MUST be Chronicle-current. Image tags/versions are re-confirmed at build time (sources: docs.bsvblockchain.org Teranode; github.com/bsv-blockchain/teranode, teranode-quickstart).

`SYS-NODE-002` The forked database and services **MUST** talk to **Teranode** over its RPC interface (transaction submission, chain/UTXO/asset queries) and its event/notification interface (new-block/new-tx), and **MAY** consume Teranode's microservice APIs (e.g. its asset/UTXO, block-assembly, and propagation services) directly where that is the supported integration path. The node is the source of truth for chain state; services hold no authority the chain does not confirm. Because Teranode is a distributed microservices system rather than a single `bitcoind`, the **exact** RPC methods, service endpoints, and event-topic names MUST be taken from the current Teranode documentation at build time — they are not assumed here.

`SYS-NODE-003` Regtest **MUST** support: instant block generation for deterministic tests, a funded coinbase for fees, and full teardown/rebuild. All adversarial and lifecycle tests (Section 12) run here first.

---

## 4. Native BSV Encoding (No P2SH, No OP_RETURN)

`SYS-ENC-001` Object/state/token data **MUST** be carried by one of, and **only** one of: (a) large pushdata inside a **spendable** locking script, placed in an unexecuted branch (`OP_FALSE OP_IF <data…> OP_ENDIF`) or pushed and dropped (`<data> OP_DROP …`) ahead of the authorisation opcodes, so the output stays spendable; or (b) a hash on-chain with the body held off-chain (forked Postgres / DHT, Section 10.2). `OP_RETURN` is **forbidden** (`SYS-CON-008`). The choice per artefact is recorded in its policy and justified, not defaulted. Pushdata/script-size limits and the data-in-spendable-output pattern are resolved in Section 14.1; only the in-force miner policy values are confirmed at build.

`SYS-ENC-002` Authorisation **MUST** be native: `OP_CHECKSIG` (single controller) or bare `OP_CHECKMULTISIG` (N-of-M), directly in the locking script — **no P2SH wrapper**. `OP_CHECKMULTISIG` semantics (required leading dummy item; signatures in public-key order; Chronicle NULLDUMMY/NULLFAIL relaxations for tx version > 1) are resolved in Section 14.1.

`SYS-ENC-003` Time-based recovery branches **MUST** use `OP_CHECKLOCKTIMEVERIFY` / `OP_CHECKSEQUENCEVERIFY`, as in the CTO substrate.

`SYS-ENC-004` Signatures **MUST** use a sighash that commits to the successor output(s) so a relay cannot redirect a state transition. **Resolved (1 Jun 2026):** on BSV `SIGHASH_FORKID` **must always be set** (replay protection); the baseline digest is the BIP143 algorithm. To bind all outputs (including the successor), use `SIGHASH_ALL | SIGHASH_FORKID`. The Chronicle upgrade additionally reinstates the **Original Transaction Digest Algorithm (OTDA)** as opt-in via the `CHRONICLE [0x20]` sighash bit and, for transaction version > 1, relaxes malleability rules (NULLFAIL, NULLDUMMY, MINIMALIF, clean-stack). The build uses `SIGHASH_ALL|FORKID` unless a documented reason selects OTDA. (Source: docs.bsvblockchain.org SIGHASH flags; bitcoin-sv Chronicle release notes.)

`SYS-ENC-005` A single canonical, versioned, length-prefixed binary layout **MUST** define every on-chain data field (magic, version, object/stream id, sequence, prev-hash, ECDH-HMAC tag, token/state payload or its hash). Encoder/decoder are covered by round-trip and rejection tests.

---

## 5. The ECDH-HMAC Hash-Chain Transaction Log (Keystone)

This is the core mechanism. It is grounded in EP3860037A1 (read in full): deterministic sub-key derivation from a public Message and a common secret computed by ECDH on the derived sub-keys, with chain-wide discoverability of transactions carrying an element's sub-key.

### 5.1 Per-change common secret

`SYS-HMAC-001` For every accounting-relevant SQL change `c` (an INSERT, UPDATE, or DELETE affecting one or more fields), the system **MUST** form a canonical **Message** `M(c)` identifying the change: `M(c) = canonical(table_id, row_id, column_id, op, seq, prev_txid)`. `M(c)` carries no secret and **MAY** travel on an insecure channel (EP3860037A1, claims 9–10).

`SYS-HMAC-002` The system **MUST** compute `GV = SHA-256(M(c))` and derive party sub-keys deterministically from each party's master key and `GV` (`V2 = V1 + GV mod n`; `P2 = P1 + GV·G`), then compute the **common secret** `CS = V2_writer · P2_counterparty = V2_counterparty · P2_writer` by ECDH on the derived sub-keys (EP3860037A1, claims 1–5). The counterparty is the reconciling party for that ledger relationship (e.g., the trading partner, or a designated auditor key for single-party books).

`SYS-HMAC-003` The HMAC key **MUST** be `K_hmac = HKDF(domain="TE/hmac/v1", ikm=CS, salt=table_id||row_id||column_id, info=seq)`. `CS` itself **MUST NOT** be used directly as the HMAC key (domain separation).

### 5.2 The on-chain tag

`SYS-HMAC-004` The system **MUST** compute `tag(c) = HMAC-SHA256(K_hmac, canonical(change_image))`, where `change_image` is the committed representation of the field value(s) changed (the value itself when confidentiality permits, or a blinded commitment to it when it does not — see `SYS-HMAC-009`).

`SYS-HMAC-005` `tag(c)` **MUST** be placed **in the spendable locking script** as pushdata (per `SYS-ENC-001`; never `OP_RETURN`) of the transaction that records change `c`. The tag's presence on chain, in a spendable output, is mandatory (`SYS-CON-003`, `SYS-CON-008`).

### 5.3 Discoverability

`SYS-HMAC-006` Because `M(c)` is reconstructable from the (public) change identity and `CS` is regenerable from master keys + `GV`, any authorised party **MUST** be able to recompute `tag(c)` and locate the recording transaction by scanning the chain for that tag/sub-key (EP3860037A1, claim 18). The system **MUST** maintain an index from `(table_id,row_id,column_id,seq)` to `txid`, rebuildable purely from the chain.

`SYS-HMAC-007` A party lacking the master key or the relationship **MUST NOT** be able to recompute `tag(c)`, so the tag does not leak which row/column changed to outsiders, while remaining fully discoverable to the parties (EP3860037A1, claim 10 — the regeneration record need not be stored privately).

### 5.4 The hash chain

`SYS-HMAC-008` Each recording transaction **MUST** include `prev_txid` (or `prev_hash`) of the immediately preceding entry in its stream, so the entries form a **hash chain**: a linked, append-only TX log. The system **MUST** define the streams (at minimum one per ledger; optionally one per account or per table) in policy. Walking a stream from genesis **MUST** reconstruct the full ordered history of changes for that stream, and the union of streams **MUST** reconstruct the entire ledger (`SYS-CON-004`).

`SYS-HMAC-009` Where a field value is confidential, the on-chain `change_image` **MUST** be a blinded commitment (CTO commitment), and the plaintext lives in the forked Postgres under the CTO confidentiality rules; the HMAC tag still binds the committed value, so integrity and discoverability hold without disclosing the value.

`SYS-HMAC-010` The chain entry, not the database row, is the **authoritative third entry**. On conflict between a local DB row and the chain, the chain wins after the policy confirmation depth; the DB is a fast local materialisation of the authoritative chain log.

### 5.5 What this proves and does not

`SYS-HMAC-011` The mechanism proves: (a) **integrity** — any tampering with a recorded value breaks its HMAC tag; (b) **completeness/ordering** — a missing or reordered entry breaks the hash chain; (c) **non-repudiation** — the recording transaction is signed by the writer's key and timestamped by the chain; (d) **bilateral verifiability** — both parties (or party + auditor) independently recompute every tag. It does **not** prove that a party never saw a value it legitimately held (CTO Statement B carries over).

---

## 6. Forked PostgreSQL with Native BSV

### 6.1 Goal

`SYS-PG-001` The deliverable **MUST** be a fork of PostgreSQL (the approved base) in which committed accounting writes are automatically mirrored to BSV per Section 5, with the on-chain mechanics invisible to the SQL user (`SYS-CON-005`). **Resolved (1 Jun 2026):** fork **PostgreSQL major version 18** (latest stable 18.4, May 2026). PostgreSQL is distributed under the **PostgreSQL License**, a permissive OSI-approved licence that permits forking, modification, and redistribution; the fork MUST preserve the upstream copyright/licence notice. The implementer confirms the exact 18.x minor at build time.

### 6.2 Write interception

`SYS-PG-002` The fork **MUST** intercept committed changes deterministically. The candidate mechanisms, to be decided in `SYS-DECIDE-002`, are: (a) logical decoding / WAL output plugin (read committed changes from the write-ahead log); (b) a native C extension with commit hooks; (c) deeper in-core hooks in the commit path. The chosen mechanism **MUST** capture every accounting-relevant change exactly once, in commit order, with the table/row/column identity needed for `M(c)`.

`SYS-PG-003` Capture **MUST** be transactional with the SQL commit: either the change is journalled to the local outbox within the same database transaction (and the BSV transaction is built/broadcast asynchronously by a writer process with at-least-once delivery and idempotent dedhuplication by `M(c)`), or the commit blocks on chain acceptance (synchronous mode). Both modes **MUST** be offered and selected per deployment (`SYS-DECIDE-003`); the asynchronous mode is the default for throughput, with the outbox guaranteeing no lost entry.

### 6.3 DB↔chain mapping

`SYS-PG-004` The fork **MUST** maintain the index of `SYS-HMAC-006` and **MUST** be able to **extract** a recorded value from the chain back into the database (EP3860037A1, claim 17; US20220253835A1 — database keyed/mapped from scriptPubKey-derived values). Cold-rebuild of the entire database from the chain alone **MUST** be supported: given the master keys and the stream genesis, the fork reconstructs every row by walking the hash chain and recomputing tags.

### 6.4 SQL-simple surface

`SYS-PG-005` A user with standard SQL knowledge **MUST** be able to operate the system without writing any blockchain code. Configuration (which tables are journalled, the counterparty/auditor key per relationship, the stream layout, the cash/token bindings) **MUST** be expressed as SQL DDL extensions or catalog tables, e.g. a `triple_entry` table property and per-relationship key registration. Reads, writes, joins, and transactions behave as normal SQL; the on-chain third entry happens underneath.

`SYS-PG-006` The fork **MUST** expose verification as SQL-callable functions (e.g. `te_verify(table, row)` returns whether the on-chain tag matches the current value and the entry's position in the hash chain), so an auditor uses SQL to verify the chain, not a separate tool.

### 6.5 Failure, reorg, and recovery

`SYS-PG-007` On a chain reorg, the fork **MUST** re-evaluate affected entries against the new canonical chain and surface any divergence; accounting-final reads **MUST** gate on the policy confirmation depth (`SYS-DECIDE-004`). The outbox and idempotent keying **MUST** ensure no double-recording and no lost entry across node restarts, fork restarts, and broadcast failures.

---

## 7. Triple-Entry Accounting Model

Grounded in Ijiri (1986), McCarthy (1982), and Cai (2019), read in full. No nChain patent covers triple-entry; this section cites the academic sources, not the patents.

### 7.1 The three entries

`SYS-TE-001` The system **MUST** implement triple-entry as: each party keeps its own **double-entry** books (debits and credits) in its forked-Postgres ledger (entries one and two, private), and each economically relevant entry is bound to a **third entry** on BSV — the ECDH-HMAC hash-chain node of Section 5 — which is the shared, immutable, cryptographically-signed record both parties reconcile against. The third entry is the receipt that is itself the transaction.

`SYS-TE-002` For a bilateral transaction, both parties' relevant entries **MUST** reference the **same** third-entry transaction (or linked entries in the shared stream), so that the two private double-entry books are provably reconcilable to one shared record. Disagreement between the parties' books and the shared chain entry is detectable by `te_verify` (`SYS-PG-006`).

### 7.2 Shared semantics

`SYS-TE-003` "Shared" **MUST** mean: the third-entry stream for a relationship is visible to and verifiable by both parties (and any authorised auditor), each holding the keys to recompute tags, while values remain confidential to outsiders per `SYS-HMAC-007`/`SYS-HMAC-009`.

### 7.3 REA grounding

`SYS-TE-004` Where the ledger models economic exchanges, it **SHOULD** follow the REA pattern (McCarthy 1982): Resources, Events, Agents. A logistics consignment (resource), its transfer (event), and the trading parties (agents) map to goods tokens (D), EDI document transitions (B), and controller keys respectively, with each event journalled as a third entry.

### 7.4 Blockchain-triple-entry grounding

`SYS-TE-005` The design **MUST** reflect the consolidation Cai (2019) describes — that a shared, tamper-evident record reconciling counterparties' books is the substance of blockchain triple-entry — and **MUST NOT** overstate it: the chain guarantees integrity, ordering, and non-repudiation of recorded entries, not the truthfulness of what a party chose to record. This limitation **MUST** be stated to users.

---

## 8. Tokenisation Core (a Definable Token for Any Item or Money Unit)

Grounded in EP3748903A1 — the *Universal Tokenisation System* (read in full) — re-expressed in native BSV (no P2SH per `SYS-CON-002`). The token primitive is general: it represents **any item or any unit of value**, with the type **definable** by the issuer; cash and goods are instances of it.

`SYS-TOK-001` A token **MUST** be represented by metadata in a transaction's locking script together with at least one public key, the metadata being a representation of, or reference to, a tokenised entity (EP3748903A1, claims 1–2, 13). In native BSV the metadata is carried by pushdata inside a spendable locking script (`SYS-ENC-001`), **not** a P2SH redeem script and **never** an `OP_RETURN`.

`SYS-TOK-002` The tokenised entity **MUST** be storable on or off chain (EP3748903A1, claim 4); off-chain storage is the forked Postgres ledger and/or the DHT (Section 10.2, grounded in US11210372).

`SYS-TOK-003` Value binding **MUST** follow EP3748903A1 claim 9: `satoshi_quantity (B1) = f(token_value TV1, pegging_rate PR1)`, with a minimum threshold (claim 10). Authorisation **MUST** be N-of-M via bare `OP_CHECKMULTISIG` (claim 7), native (no P2SH).

`SYS-TOK-004` Token transfer **MUST** be a UTXO-lineage transfer on the CTO substrate: spend the current token UTXO, create the successor bound to the new controller, no residual prior-controller path except time-locked recovery.

`SYS-TOK-005` **Definable token type.** The system **MUST** provide a **token-definition schema** by which an issuer defines a new token type without code changes: a unique type id; a human label; the unit it denominates (a fiat/CBDC currency code, a crypto/stablecoin unit, a commodity unit, a goods SKU, a loyalty point, or any definable unit); divisibility/precision; supply policy (fixed, mint-on-demand, burnable); the backing/pegging model (`SYS-TOK-003`); the controller/issuer key model; and the confidentiality policy. EP3748903A1 grounds this: a token is metadata representing or referencing **any** tokenised entity, stored on or off chain (claims 1–2, 4), with value set by `token_value × pegging_rate` (claim 9). Every token in the system (cash, CBDC-linked, stablecoin-linked, goods, document) is an instance of this one definable primitive.

`SYS-TOK-006` **External linkage / interoperability.** A token definition **MAY** declare an external linkage so the token represents or is exchangeable for value held in another system: (a) **peg/back** to an external unit (another BSV token-coin or token protocol, a stablecoin, or a **central-bank digital currency (CBDC)**) via the pegging-rate mechanism plus a named **oracle** for the rate and a named **custodian/issuer** for the backing; or (b) **bridge** to an external token system through a defined adapter that locks/mints on one side against the other. The external system's own interfaces (a CBDC rail, a stablecoin issuer's API, another token protocol) are integrated by **adapters** and are **not** specified here beyond the adapter contract; the spec does not assume any external system's behaviour. On-chain, the linked token is still the native definable token of `SYS-TOK-005`; the linkage is metadata + adapter, never a P2SH or `OP_RETURN` construct.

`SYS-TOK-007` **Exchangeability.** Any two token instances **MUST** be exchangeable through an atomic on-chain swap of their UTXO lineages (deliver-versus-deliver), so a unit of one definable token can be traded for a unit of another (goods-for-cash, cash-for-CBDC-linked, token-for-token), each leg journalled as a third entry. This is the general form of the delivery-versus-payment linkage of `SYS-LOG-008`.

### 8.1 Cash — a money-unit instance of the definable token

`SYS-CASH-001` A cash token is an instance of the definable token (`SYS-TOK-005`) whose unit is a **money unit** — any fiat currency, a **CBDC**, or a token-based coin/stablecoin on BSV or elsewhere — carrying a unit identifier and a token value, with satoshi backing set by the pegging rate (`SYS-TOK-003`). **Decision (`SYS-DECIDE-001`, locked): all profiles are provided and selectable per token definition**, and a cash token **MAY** be linked to an external coin/stablecoin/CBDC per `SYS-TOK-006`:
- **Issuer-backed redeemable:** an issuer key controls mint/redeem; the token is a claim on off-chain value the issuer holds (fiat, or a CBDC/stablecoin balance); redemption burns the token.
- **Satoshi-tagged:** the token denominates underlying BSV value tagged with a unit and a pegging-rate snapshot; no external issuer.
- **Pegged / externally linked:** the token tracks an external unit (currency, CBDC, or another token coin) via an oracle-published pegging rate and, where backed, a named custodian, with a documented peg-maintenance and de-peg policy (`SYS-TOK-006`).

`SYS-CASH-002` Every cash mint, transfer, and redemption **MUST** be journalled as a third entry (Section 5) so settlement is part of the shared triple-entry record.

### 8.2 Goods tokens

`SYS-GOODS-001` A goods token **MUST** represent a consignment/item, with the detailed record (description, quantity, provenance, custody history) off-chain (forked Postgres and/or DHT) and the control/transfer on-chain. Ownership verification follows US11210372 (Section 10.2).

---

## 9. Tokenised EDI / Commercial-Document Engine

Grounded in US20220253835A1 (read in full), native BSV (no P2SH). The document *types* below are standard trade/logistics artefacts (domain facts); the *mechanism* (DFA-state-as-UTXO, discoverability) is the patent's, re-expressed natively.

`SYS-EDI-001` Each commercial-document lifecycle **MUST** be a deterministic finite automaton whose states are UTXOs; a transition spends the current state UTXO and creates the next (US20220253835A1 — DFA states incarnated as UTXOs; a transition spends one state's UTXO and creates the next). Document content/metadata is carried per `SYS-ENC-001`; confidential fields are CTO-committed.

`SYS-EDI-002` The system **MUST** implement, as DFAs, the complete commercial-and-logistics document set, each with an explicit state set, event alphabet, and transition table (per the worked DFA method of US20220253835A1, which demonstrates state set, input alphabet, transition matrix, origination and completion transactions, and bounded cost). The mandatory set:

**Trade / payment documents**
- Request for quotation (RFQ) and quotation
- Purchase order (PO) and order acknowledgement / order change
- Commercial invoice (**invoice note**)
- **Payment note** (settlement instruction / remittance) — links to the cash package (Section 8.1)
- Credit/debit note

**Logistics / transport documents**
- Despatch advice / Advance Shipping Notice (ASN / DESADV)
- **Bill of lading (B/L)** — title document; **the consignment token itself is the negotiable B/L** (Section 10.3)
- Sea/air waybill; road consignment note (CMR); rail consignment note
- Packing list
- Booking confirmation / transport instruction (IFTMIN-class)
- Arrival notice
- **Proof of delivery (POD)**

**Regulatory / assurance documents**
- Certificate of origin
- Customs / import-export declaration
- Inspection / quality / condition certificate
- Insurance certificate

`SYS-EDI-003` Every document state transition **MUST** be journalled as a third entry (Section 5) and **MUST** be discoverable on chain by the US20220253835A1 method re-expressed natively: derive a search key from the state metadata, scan for the matching UTXO, extract, determine the current state. The forked-Postgres index (`SYS-HMAC-006`) is the materialised form of that search.

`SYS-EDI-004` Documents **MUST** cross-reference by `object_id`: an invoice references its PO and its consignment(s); a payment note references its invoice; a POD references its B/L. The reference graph **MUST** be reconstructable from chain alone.

`SYS-EDI-005` **Decision (`SYS-DECIDE-005`, locked): the standards bridge IS provided**, as a configurable, optional component enabled per trading partner. The system **MUST** ship a bidirectional bridge mapping real EDI standard messages ↔ the on-chain document DFA states (`SYS-EDI-002`):
- **ANSI X12**: 850 (purchase order), 855 (PO acknowledgement), 856 (ASN/despatch), 810 (invoice), 820 (payment/remittance), 214 (transport status), 990/210 (freight). 
- **UN/EDIFACT**: ORDERS, ORDRSP, DESADV, INVOIC, REMADV, IFTMIN (transport instruction), IFTSTA (transport status), PAYORD.

`SYS-EDI-006` The bridge **MUST** be a pure translation layer: inbound, it parses a standard message, validates it, and drives the corresponding DFA transition (journalled as a third entry); outbound, it serialises current on-chain document state into the standard message for a partner that transacts in X12/EDIFACT. The on-chain DFA remains the source of truth; the bridge holds no authority the chain does not confirm. It is enabled/disabled and configured per partner (which standard, which version, which message subset), and being optional it MUST be omittable without affecting the core DFA engine.

### 9.6 Document rendering and paper copies (PDF)

`SYS-DOC-001` Any commercial or logistics document (Section 9.2) — and the **bill of lading** in particular — **MUST** be renderable, on demand, to a complete human-readable **PDF paper copy** assembled deterministically from three sources: (i) the document's on-chain DFA state and lineage (Sections 5, 9); (ii) its off-chain detail record (forked Postgres and/or DHT, Section 10.2); and (iii) its SPV inclusion proof (BURI + Merkle proof, Section 11.1). The PDF **MUST** contain every field needed to read and act on the document as if it were the conventional paper instrument (parties, dates, line items, quantities, amounts, terms, references to PO/invoice/consignment, signatures/keys, and status).

`SYS-DOC-002` The PDF **MUST** embed its own verifiability: the document's `object_id`, the current-state transaction reference as a **BURI** (`SYS-PROOF-003`), and the Merkle/SPV proof, rendered both as text and as a scannable code (e.g. a QR of the BURI), so that a holder of the paper copy — or a verifier scanning it — can independently confirm against block headers that the document and its state exist on chain, without trusting the issuer or the database operator (`SYS-PROOF-005`).

`SYS-DOC-003` Rendering **MUST** be deterministic and reproducible: the same document at the same state **MUST** produce a byte-stable PDF (fixed templates, fixed field order, no nondeterministic timestamps in the body beyond the recorded state), and the rendering function **MUST** be exercised by tests asserting that the PDF's embedded `object_id`/state/proof match the chain. The implementation uses the `pdf` skill conventions; confidential fields follow the CTO confidentiality rules and are included only when the renderer holds the entitlement keys.

`SYS-DOC-004` **Single-original / negotiability honesty.** For a negotiable bill of lading, the **on-chain token remains the single source of title** (`SYS-LOG-006`); a rendered PDF is a **verifiable representation, not a second negotiable original**, and **MUST** be marked as such (e.g. "COPY — title held on-chain; current holder verifiable via the embedded BURI"). The system **MUST NOT** present a PDF as transferring title. Where a controlled single-paper-original workflow is genuinely required (e.g. a counterparty that cannot transact on chain), it **MUST** be modelled explicitly as a state transition that locks the token (a `PAPER_ISSUED` state) so the token and a paper original cannot both be live at once; this is an explicit, audited transition, not a side effect of rendering.

`SYS-DOC-005` The renderer **MUST** be invocable from the SQL surface (e.g. `te_render_pdf(object_id)`), consistent with `SYS-CON-005`, returning or writing the PDF, so producing a paper copy is "just SQL" like the rest of the system.

---

## 10. Logistics / Goods Package (Complete)

Grounded in US11210372 (read in full) for goods records and ownership, EP3748903A1 for the token, US20220253835A1 for lifecycle, and EP3860037A1 for confidentiality of records — all re-expressed natively (no P2SH, no redeem script).

### 10.1 Consignment lifecycle (DFA)

`SYS-LOG-001` A consignment **MUST** be a goods token (Section 8.2) whose lifecycle is a DFA per Section 9. The mandatory state set: `CREATED → BOOKED → PICKED_UP → IN_TRANSIT(leg_k) → CUSTOMS_HELD/CLEARED → INSPECTED → CUSTODY_TRANSFER(party_k) → DELIVERED → ACCEPTED/REJECTED → SETTLED`, with `DISPUTED` and time-locked `RECOVERED` branches from the CTO substrate. Each economically or custody-relevant transition is a UTXO-lineage transfer journalled as a third entry (Section 5).

### 10.2 Goods records and ownership (grounded: US11210372)

`SYS-LOG-002` A goods record (description, HS/commodity code, quantity, weight, provenance, custody history, condition data, attached documents) **MUST** be stored in a **separate storage resource — a Distributed Hash Table (DHT)** — not bloated on chain (US11210372: the asset's data and its location live in the DHT as a key-value entry).

`SYS-LOG-003` The on-chain consignment metadata **MUST** carry a hash `H2` representative of the goods-record details and the owner/custodian public key (US11210372). `H2` **MUST** be the **key of the DHT key-value pair**; the DHT value holds the record data `D1`, a first hash `H1` of the body, and an identifier indicating the **location** of the full record (US11210372, claim 1–2). The record is structured as **header + body**, the header carrying the hash of the body (claims 3–4), so integrity of the off-chain body is anchored on chain.

`SYS-LOG-004` **Ownership/custody verification MUST** follow US11210372: derive the parties' second public keys from their public keys and a generator value `GV`, compute the **common secret `CS`** (ECDH on the derived keys — the same construction as EP3860037A1 §5), and **verify control by matching the on-chain controller key against the owner key registered in the DHT entry** (compare `PU2` with `P2`; a match verifies ownership). The goods record **MUST** be encrypted under `CS` so only the entitled parties read it (US11210372), consistent with the CTO confidentiality boundary.

`SYS-LOG-005` Because verification reduces to a key match plus an `H2`→DHT lookup, any authorised party **MUST** be able to verify current ownership/custody of a consignment from the chain plus the DHT, with no trusted intermediary.

### 10.3 Bill of lading as the token (title transfer)

`SYS-LOG-006` Where the consignment is governed by a negotiable bill of lading, the **goods token MUST be the electronic B/L**: transferring the token (a UTXO-lineage transfer on the CTO substrate) **is** the transfer of title to the goods. Endorsement/transfer of the B/L is therefore a single on-chain transfer, journalled as a third entry, with the new holder's key bound and no residual prior-holder spend path except time-locked recovery. This removes paper B/L handling while preserving negotiability. A **paper copy** of the B/L, when needed, is produced per Section 9.6 as a verifiable PDF that does not itself transfer title (`SYS-DOC-004`).

### 10.4 Multi-party, multi-leg custody

`SYS-LOG-007` The custody chain (shipper → carrier(s) → freight forwarder → customs → consignee) **MUST** be representable as successive `CUSTODY_TRANSFER` transitions, each re-keying control to the next custodian and journalling a third entry, so the **complete chain of custody is reconstructable from chain + DHT**. Each leg **MAY** carry its own condition/inspection event appended to the DHT record (with the header hash updated on chain) so tampering with custody history is detectable.

### 10.5 Settlement linkage

`SYS-LOG-008` Delivery acceptance (`ACCEPTED`) **MUST** be linkable to release of a cash token settlement (Section 8.1) and to the invoice/payment-note documents (Section 9), so that delivery-versus-payment can be enforced: the payment note transitions to settled on the same evidence (POD + acceptance) that advances the consignment, all journalled as third entries in the shared stream.

### 10.6 Integrity verification of goods records and document bodies (grounded: GB2558485A / US10579779B2)

`SYS-LOG-009` Every off-chain record anchored on chain — a goods record (Section 10.2), an EDI/logistics document body (Section 9), or any DHT-stored asset — **MUST** be integrity-verifiable by the method of GB2558485A: from the on-chain metadata `M` of the record's transaction, determine an indication of the DHT entry; compute a fresh hash **H3** over the current asset/record body; read the stored hash **H4** from the DHT entry; the record's integrity holds **iff `H3 == H4`** (GB2558485A method 900, steps 910–960, read in full from US10579779B2).

`SYS-LOG-010` Records **MUST** be structured as **header + body**, where the header carries a hash of the body (and may carry the DHT key `H2`), so that the on-chain anchor commits to the body and any later alteration of the body is detected by `SYS-LOG-009` (US10579779B2 — header comprises a hash value of the body). The DHT entry is keyed by `H2` (a hash representative of the record details), with the value holding the record data, the body hash, and the record's location (consistent with the ownership key-value structure of Section 10.2).

`SYS-LOG-011` Integrity (`SYS-LOG-009`, hash match) and **ownership/custody** (Section 10.2, controller-key vs DHT-registered-key match) are **distinct, complementary checks** and **MUST** both be exposed: a consignment or document can be confirmed as both unaltered (integrity) and under the asserted holder's control (ownership). Where confidentiality applies, the body is encrypted under the ECDH common secret (Section 5) before hashing/storage, exactly as the family teaches (common-secret derivation + symmetric encryption of the asset).

`SYS-LOG-012` The integrity check **MUST** back the PDF paper copy (Section 9.6): a rendered document's embedded references let a verifier recompute `H3` over the rendered body and confirm it against the on-chain-anchored `H4`, so a paper copy is provably faithful to the on-chain record (strengthens `SYS-DOC-002`).

---

## 11. Verification, Custody, Addressing, and Computation (Grounded)

All four clusters below are now read in full; the v1/v2 stubs are replaced with grounded requirements, re-expressed in native BSV (no P2SH).

### 11.1 Existence / inclusion proofs (grounded: WO2022100946A1, WO2022214264A1, WO2025119666A1)

`SYS-PROOF-001` Any party (counterparty, auditor, regulator) **MUST** be able to prove that a given third-entry transaction — and therefore a recorded SQL change and its ECDH-HMAC tag (Section 5) — exists on chain via SPV Merkle proof, **without running a full node** (WO2022100946A1, claim 1).

`SYS-PROOF-002` A **Merkle-proof service entity** **MUST** be providable that stores transaction identifiers and per-block subsets with their block headers (not the full chain, not publishing blocks), and on request returns a Merkle proof plus the leaf index proving a target data item exists in a target transaction in a block (WO2022100946A1, claims 1, 4, 17–20). The forked-Postgres index (`SYS-HMAC-006`) **MAY** act as, or be backed by, such an entity.

`SYS-PROOF-003` Inter-transaction references — the hash-chain back-reference of `SYS-HMAC-008` and the cross-document references of `SYS-EDI-004` — **MUST** be expressible as a **BURI** (Blockchain Uniform Resource Indicator): a delimiter-separated string carrying a block identifier, a transaction identifier, and Merkle-proof portions (Merkle index + proof hashes), SPV-verifiable against a Merkle root **without accessing the block payload** (WO2022214264A1, claims 1, 3–4). A referencing transaction **MAY** embed a BURI in an output to point verifiably at a prior entry (WO2022214264A1, claim 9).

`SYS-PROOF-004` For compact, privacy-preserving proof publication, the system **MAY** publish on chain the data items used to build a Merkle proof, split the proof into indexed portions, and apply a **homomorphic / elliptic-curve commitment** to Merkle-tree nodes at a chosen level (a sum of EC points representing those nodes) to enable verification of a data item's presence (WO2025119666A1, claims 1–4, 8–11).

`SYS-PROOF-005` `te_verify` (`SYS-PG-006`) **MUST** export, for any row/change, a self-contained SPV proof (BURI + Merkle proof) verifiable by a third party with only block headers — making the triple-entry ledger auditable by anyone holding the relationship keys, without trusting the database operator.

### 11.2 Threshold custody and signing (grounded: US11671255, EP3259724B1)

`SYS-CUST-001` Cash tokens, goods tokens, and ledger master keys **MUST** be protectable by N-of-M threshold control in which the private key is **never reconstructed**: a threshold of share-holders produce partial signatures that combine into one valid signature (US11671255, claim 1), **robust against faulty or missing shares** via the error-locator-polynomial reconstruction (US11671255 — error-locator + second polynomial method). This is the native realisation of the CTO Tier-S profile.

`SYS-CUST-002` Key shares **MUST** be distributable and recoverable by the loss-resistant method of EP3259724B1: split the key (or a key-access element) into shares; transmit each share encrypted under the **ECDH common secret** of Section 5 (EP3259724B1, claim 1 — the same derivation); and store **at least three shares at separate locations, at least one in a backup/safe-storage facility** (EP3259724B1, claims 1, 3).

`SYS-CUST-003` Threshold authorisation **MUST** be expressed natively (bare `OP_CHECKMULTISIG`, or a single `OP_CHECKSIG` over a threshold-signed key — no P2SH). The choice between on-chain multisig and off-chain threshold-signing-to-one-key is recorded per deployment.

### 11.3 Overlay addressing and routing (grounded: EP4046048B1)

`SYS-OVL-001` The hash-chain streams (Section 5.4), document graphs (Section 9), and token lineages (Section 8) **MUST** be modelled as an **overlay network on BSV data-storage transactions**: a graph of nodes (transactions) and edges (links), with overlay content in transaction payloads, and **each node carrying a key that signs the input of its child transaction** to authorise writing the child (EP4046048B1, claim 1) — the lineage is a key-signed transaction graph.

`SYS-OVL-002` Overlay keys **MUST** be derivable by a **child-key-derivation (CKD) hierarchy whose key structure mirrors the overlay graph** (EP4046048B1, claim 1 — a hierarchical set of keys with the same graph structure as the overlay). This gives deterministic, structure-aligned addressing of every stream, document, and token from a seed, dovetailing with the common-secret derivation of Section 5.

`SYS-OVL-003` Discovery and routing of an entity (stream, document, token, counterparty) **MUST** use its overlay address (graph position + derived key), with BURIs (`SYS-PROOF-003`) as the verifiable edge references between nodes.

### 11.4 Computation layer (grounded: US20240364498A1)

`SYS-COMP-001` For computations beyond a DFA transition — adjudicating a disputed logistics condition, computing a customs valuation, evaluating an oracle-fed predicate — the system **MAY** use the staked-proposer/challenge market of US20240364498A1: a requester posts a task with a bounty (first digital asset); a proposer commits a solution by hash and stakes a second digital asset; on challenge, both assets are placed under **threshold control of a group** (released when a threshold of the group signs); the challenge is resolved by selecting a solution, and assets are distributed per the result (US20240364498A1, claim 1).

`SYS-COMP-002` A resolved computation result **MUST** be feedable as an input event into the relevant DFA (Section 9), so a disputed or computed outcome advances the document/consignment lifecycle and is journalled as a third entry. The threshold-controlled escrow of `SYS-COMP-001` reuses the custody mechanism of `SYS-CUST-001`.

---

## 12. Testing and Verification (Docker regtest first)

`SYS-TEST-001` All tests **MUST** run end-to-end against the Dockerised full BSV node on regtest before testnet/mainnet (`SYS-CON-006`).

`SYS-TEST-002` The suite **MUST** include: ECDH-HMAC known-answer vectors (derive `CS`, `K_hmac`, `tag` from fixed inputs); hash-chain integrity tests (tamper → tag mismatch; drop/reorder → chain break); cold-rebuild test (reconstruct the entire DB from the chain + master keys and assert equality); SQL-surface tests (ordinary SQL produces correct on-chain entries with no user blockchain code); reorg tests; tokenisation lifecycle (mint/transfer/redeem for each cash profile and goods); EDI DFA lifecycle per document; **PDF paper-copy rendering** (deterministic byte-stable output; embedded object_id/state/BURI/Merkle proof match the chain; SPV-verifiable from headers alone; B/L copy is marked non-negotiable per `SYS-DOC-004`); and the CTO substrate's own adversarial suite.

`SYS-TEST-003` Cross-implementation vectors **MUST** be shared between the C (fork) and TypeScript/Go components for ECDH, HKDF, HMAC, commitment, and encoding; divergence is a release blocker.

---

## 13. Repository and Phased Delivery

`SYS-REPO-001` Monorepo (illustrative):
```
/te-bsv
  /node-docker            # Dockerised full BSV node + compose [VERIFY node image]
  /pg-fork                # forked PostgreSQL + native BSV (C)
    /capture              # write interception (WAL/extension/hooks)
    /bsv-native           # key derivation, ECDH-HMAC, tx build/broadcast, hash chain
    /sql-surface          # DDL extensions, catalog, te_verify()
  /crypto-core            # shared ECDH/HKDF/HMAC/commitment (C + TS/Go), KAT vectors
  /tokenisation           # definable token primitive (any item/unit) + cash/CBDC linkage + goods, native BSV
  /edi-dfa                # commercial-document DFAs (PO/invoice/payment/shipping)
  /edi-bridge             # optional X12/EDIFACT <-> on-chain DFA translation (per partner)
  /doc-render             # deterministic PDF paper copies (pdf skill) + BURI/QR embed
  /logistics              # consignment lifecycle + DHT goods records (US11210372)
  /sdk-ts                 # client SDK
  /services-go            # indexer, relay, broadcaster glue to node
  /spec                   # this document + requirements + VERIFY-LOG + DECISIONS
  /tests                  # regtest e2e, KAT, adversarial, cold-rebuild
```

`SYS-PHASE-001` Phased, each gated on its exit criteria:
- **Phase 0 — Spec freeze + decisions.** Resolve `SYS-DECIDE-*`; assign every `[VERIFY]`; pin node image and PG version.
- **Phase 1 — Crypto core + KAT.** ECDH common-secret (EP3860037A1 method), HKDF, HMAC, commitment, encoding; cross-impl vectors green.
- **Phase 2 — Node + hash-chain log.** Dockerised node; build/broadcast a hash-chained ECDH-HMAC TX stream on regtest; discoverability + cold-rebuild of a toy stream.
- **Phase 3 — PG fork.** Write interception + outbox + async/sync modes + `te_verify` + cold-rebuild of a real schema. Exit: ordinary SQL produces a correct, verifiable on-chain triple-entry log.
- **Phase 4 — Tokenisation.** Cash (3 profiles) + goods, native BSV, journalled. 
- **Phase 5 — EDI DFA + logistics.** Document DFAs and consignment lifecycle; interoperation (invoice ↔ cash ↔ goods).
- **Phase 6 — Verification/custody/addressing/computation integrations.** Inclusion proofs (11.1), threshold custody (11.2), overlay addressing (11.3), computation layer (11.4) — all now grounded in patents read in full.
- **Phase 7 — Hardening.** Reorg, HA, security review, testnet.

`SYS-PHASE-002` No phase begins before the prior phase's exit criteria pass.

---

## 14. Open Decisions and Verify Register

`SYS-DECIDE-001` **Cash / token model**: **Resolved** — a general **definable token** (`SYS-TOK-005`) for any item or money unit; all three cash profiles provided and selectable per token definition; external linkage to token-coins/stablecoins/**CBDC** supported (`SYS-TOK-006`).
`SYS-DECIDE-002` **Write-interception mechanism**: WAL logical decoding vs C extension vs in-core hooks.
`SYS-DECIDE-003` **Journalling mode**: async outbox (default) vs synchronous commit-on-chain.
`SYS-DECIDE-004` **Confirmation depth** for accounting finality, per use case.
`SYS-DECIDE-005` **EDI standards bridge**: **Resolved** — provided as a configurable, optional, per-partner component (X12 850/855/856/810/820/214/990/210; EDIFACT ORDERS/ORDRSP/DESADV/INVOIC/REMADV/IFTMIN/IFTSTA/PAYORD), `SYS-EDI-005`/`SYS-EDI-006`.
`SYS-DECIDE-006` **Stream granularity**: per-ledger vs per-account vs per-table hash chains.
`SYS-DECIDE-007` **Counterparty/auditor key model** for single-party books (who is the ECDH counterparty when there is no trading partner).
`SYS-DECIDE-008` **PostgreSQL version** to fork. **Resolved:** PostgreSQL 18 (latest major), PostgreSQL License (fork-permissive).
`SYS-DECIDE-010` **BSV node**: **Resolved** — Teranode (via teranode-quickstart Docker; regtest for dev/test), `SYS-NODE-001`.
`SYS-DECIDE-009` **Controlled single-paper-original workflow** for B/L: offered (with a `PAPER_ISSUED` token-locking state, `SYS-DOC-004`) or not. Default: not offered — token is the sole title, PDF is copy-only.

`SYS-VERIFY-LIST` **Resolved (1 Jun 2026; re-confirm at build time):** see Section 14.1. **Still implementer-confirmed at build (policy, not fixed facts):** the specific miner relay policy values actually in force (script-size policy default 500 KB but miner-configurable; fees; dust thresholds) — survey target miners via their MinerID coinbase documents; and the exact node image tag/minor versions current at build.

### 14.1 Resolved technical parameters (sources current as of 1 June 2026)

These resolve the prior `[VERIFY]` gates. They are current facts, not guesses; the implementer re-confirms at build time because the ecosystem moves (Chronicle activated only April 2026; node minor versions and miner policies change).

**BSV node (Docker) — chosen: Teranode.** Teranode (Go microservices) via `teranode-quickstart` Docker supports mainnet/testnet/teratestnet/regtest; dev/test runs on regtest. SV Node (`bitcoinsv/bitcoin-sv`) is the non-chosen monolithic alternative. Chronicle upgrade mandatory; activated on mainnet 7 April 2026. Teranode RPC/service/event interface names are confirmed against current Teranode docs at build (it is a microservices cluster, not a single `bitcoind`). (docs.bsvblockchain.org Teranode; github.com/bsv-blockchain/teranode.)

**Script / consensus (post-Genesis, per bitcoin-sv consensus limits).**
- **P2SH** is prohibited as a **consensus rule** (confirms `SYS-CON-002`).
- **Pushdata / stack**: the pre-Genesis 520-byte element cap and 1000-element stack cap are replaced after Genesis by a configurable stack-memory limit; large data pushes inside spendable scripts are valid (supports `SYS-ENC-001`, no OP_RETURN needed).
- **Script size**: default relay/mine policy 500 KB per script after Genesis (miner-configurable, `maxscriptsizepolicy`); consensus is far higher.
- **Ops per script**: effectively unlimited after Genesis (`MAX_OPS_PER_SCRIPT_AFTER_GENESIS = UINT32_MAX`).
- **Multisig**: keys-per-`OP_CHECKMULTISIG` effectively unlimited after Genesis; bare `OP_CHECKMULTISIG` (P2MS) is the native N-of-M (confirms `SYS-CUST-003`). Implementation notes: the historical extra dummy stack item (the leading element consumed by `OP_CHECKMULTISIG`) is still required pre-Chronicle, and signatures must be supplied in the same order as their public keys; Chronicle relaxes NULLDUMMY (dummy may be any value) and NULLFAIL for version > 1 transactions.
- **Transaction size**: consensus 1 GB after Genesis; default policy 10 MB (miner-configurable).
- **`OP_FALSE OP_IF … OP_ENDIF` data pattern** (used by `SYS-ENC-001`): valid; Chronicle relaxes MINIMALIF so the IF argument need not be exactly 1/0.

**Sighash.** `SIGHASH_FORKID` must always be set; BIP143 digest baseline; bind successor outputs with `SIGHASH_ALL|FORKID`; OTDA available opt-in via the Chronicle `0x20` bit for tx version > 1 (`SYS-ENC-004`).

**PostgreSQL.** Fork major version **18** (latest stable 18.4, May 2026); **PostgreSQL License** (permissive, fork-allowed); preserve upstream notices (`SYS-PG-001`, `SYS-DECIDE-008`).

---

## 15. Closing

`SYS-FINAL-001` This is the contract for the build. Where reality differs from any statement here, resolve by `[VERIFY]` against the authoritative source and record it — never code around it silently. Every patent in the corpus has now been read in full (claims and description) and grounds its section, GB2558485A via its granted family text US10579779B2/WO2017195160A1; nothing is grounded on an abstract. Fabricated patent content, hidden assumptions, P2SH, and BTC constructs are each, independently, build failures.

*End of specification v1.0. Pending sections to be expanded into v2 after the remaining patents are read in full.*


---

## Appendix A — Requirements Index (traceability)

All 110 normative requirements, in document order. This index is the traceability spine: every build artefact and test maps to one or more IDs, and the build is complete when all are satisfied or explicitly waived by an `SYS-DECIDE-*`.

| ID | Requirement (summary) |
|---|---|
| `SYS-INTEG-001` | No requirement, claim, or design element in this document may be grounded in the abstract of a patent. |
| `SYS-INTEG-002` | Where a value, opcode behaviour, library API, node RPC, or ecosystem fact must be confirmed against a current authoritative source, the text carries [VERIFY]. |
| `SYS-INTEG-003` | No deployment decision is made silently. |
| `SYS-INTEG-004` | This system implements methods taught in nChain patents. |
| `SYS-SCOPE-001` | The system MUST deliver a single integrated platform with four packages sharing one substrate: |
| `SYS-SCOPE-002` | The packages MUST interoperate: an EDI invoice (B) references a cash settlement (C) and a goods consignment (D), and every state change in all packages is jo… |
| `SYS-CON-001` | BSV only. |
| `SYS-CON-002` | No P2SH. |
| `SYS-CON-008` | No OP_RETURN — at all. |
| `SYS-CON-003` | Every SQL field and every change MUST be bound to an ECDH-keyed HMAC carried in the transaction script and on chain (Section 5). |
| `SYS-CON-004` | The on-chain record MUST form a hash chain (a TX log) such that every transaction is discoverable and the full ledger history is reconstructable by traversal… |
| `SYS-CON-005` | The user-facing interface MUST be SQL. |
| `SYS-CON-006` | A full BSV node MUST run in Docker for development and testing; the system MUST operate end-to-end against it on regtest before any testnet/mainnet use (Sect… |
| `SYS-CON-007` | The cryptographic honesty boundary of the CTO substrate (forward access revocation provable; erasure of observed plaintext not provable) carries over unchanged. |
| `SYS-SUB-001` | Confidential payloads (private ledger fields, document contents, goods records) and threshold custody MUST use the CTO primitive: secp256k1 ECDH, HKDF, AEAD,… |
| `SYS-SUB-002` | The CTO model already represents an object as a single-threaded UTXO lineage with on-chain control and off-chain confidential payload. |
| `SYS-NODE-001` | The system MUST ship a Docker composition that runs a full BSV node plus the forked PostgreSQL and the service layer. |
| `SYS-NODE-002` | The forked database and services MUST talk to Teranode over its RPC interface (transaction submission, chain/UTXO/asset queries) and its event/notification i… |
| `SYS-NODE-003` | Regtest MUST support: instant block generation for deterministic tests, a funded coinbase for fees, and full teardown/rebuild. |
| `SYS-ENC-001` | Object/state/token data MUST be carried by one of, and only one of: (a) large pushdata inside a spendable locking script, placed in an unexecuted branch (OP_… |
| `SYS-ENC-002` | Authorisation MUST be native: OP_CHECKSIG (single controller) or bare OP_CHECKMULTISIG (N-of-M), directly in the locking script — no P2SH wrapper. |
| `SYS-ENC-003` | Time-based recovery branches MUST use OP_CHECKLOCKTIMEVERIFY / OP_CHECKSEQUENCEVERIFY, as in the CTO substrate. |
| `SYS-ENC-004` | Signatures MUST use a sighash that commits to the successor output(s) so a relay cannot redirect a state transition. |
| `SYS-ENC-005` | A single canonical, versioned, length-prefixed binary layout MUST define every on-chain data field (magic, version, object/stream id, sequence, prev-hash, EC… |
| `SYS-HMAC-001` | For every accounting-relevant SQL change c (an INSERT, UPDATE, or DELETE affecting one or more fields), the system MUST form a canonical Message M(c) identif… |
| `SYS-HMAC-002` | The system MUST compute GV = SHA-256(M(c)) and derive party sub-keys deterministically from each party's master key and GV (V2 = V1 + GV mod n; P2 = P1 + GV·… |
| `SYS-HMAC-003` | The HMAC key MUST be K_hmac = HKDF(domain="TE/hmac/v1", ikm=CS, salt=table_id\|\|row_id\|\|column_id, info=seq). |
| `SYS-HMAC-004` | The system MUST compute tag(c) = HMAC-SHA256(K_hmac, canonical(change_image)), where change_image is the committed representation of the field value(s) chang… |
| `SYS-HMAC-005` | tag(c) MUST be placed in the spendable locking script as pushdata (per SYS-ENC-001; never OP_RETURN) of the transaction that records change c. |
| `SYS-HMAC-006` | Because M(c) is reconstructable from the (public) change identity and CS is regenerable from master keys + GV, any authorised party MUST be able to recompute… |
| `SYS-HMAC-007` | A party lacking the master key or the relationship MUST NOT be able to recompute tag(c), so the tag does not leak which row/column changed to outsiders, whil… |
| `SYS-HMAC-008` | Each recording transaction MUST include prev_txid (or prev_hash) of the immediately preceding entry in its stream, so the entries form a hash chain: a linked… |
| `SYS-HMAC-009` | Where a field value is confidential, the on-chain change_image MUST be a blinded commitment (CTO commitment), and the plaintext lives in the forked Postgres… |
| `SYS-HMAC-010` | The chain entry, not the database row, is the authoritative third entry. |
| `SYS-HMAC-011` | The mechanism proves: (a) integrity — any tampering with a recorded value breaks its HMAC tag; (b) completeness/ordering — a missing or reordered entry break… |
| `SYS-PG-001` | The deliverable MUST be a fork of PostgreSQL (the approved base) in which committed accounting writes are automatically mirrored to BSV per Section 5, with t… |
| `SYS-PG-002` | The fork MUST intercept committed changes deterministically. |
| `SYS-PG-003` | Capture MUST be transactional with the SQL commit: either the change is journalled to the local outbox within the same database transaction (and the BSV tran… |
| `SYS-PG-004` | The fork MUST maintain the index of SYS-HMAC-006 and MUST be able to extract a recorded value from the chain back into the database (EP3860037A1, claim 17; U… |
| `SYS-PG-005` | A user with standard SQL knowledge MUST be able to operate the system without writing any blockchain code. |
| `SYS-PG-006` | The fork MUST expose verification as SQL-callable functions (e.g. |
| `SYS-PG-007` | On a chain reorg, the fork MUST re-evaluate affected entries against the new canonical chain and surface any divergence; accounting-final reads MUST gate on… |
| `SYS-TE-001` | The system MUST implement triple-entry as: each party keeps its own double-entry books (debits and credits) in its forked-Postgres ledger (entries one and tw… |
| `SYS-TE-002` | For a bilateral transaction, both parties' relevant entries MUST reference the same third-entry transaction (or linked entries in the shared stream), so that… |
| `SYS-TE-003` | "Shared" MUST mean: the third-entry stream for a relationship is visible to and verifiable by both parties (and any authorised auditor), each holding the key… |
| `SYS-TE-004` | Where the ledger models economic exchanges, it SHOULD follow the REA pattern (McCarthy 1982): Resources, Events, Agents. |
| `SYS-TE-005` | The design MUST reflect the consolidation Cai (2019) describes — that a shared, tamper-evident record reconciling counterparties' books is the substance of b… |
| `SYS-TOK-001` | A token MUST be represented by metadata in a transaction's locking script together with at least one public key, the metadata being a representation of, or r… |
| `SYS-TOK-002` | The tokenised entity MUST be storable on or off chain (EP3748903A1, claim 4); off-chain storage is the forked Postgres ledger and/or the DHT (Section 10.2, g… |
| `SYS-TOK-003` | Value binding MUST follow EP3748903A1 claim 9: satoshi_quantity (B1) = f(token_value TV1, pegging_rate PR1), with a minimum threshold (claim 10). |
| `SYS-TOK-004` | Token transfer MUST be a UTXO-lineage transfer on the CTO substrate: spend the current token UTXO, create the successor bound to the new controller, no resid… |
| `SYS-TOK-005` | Definable token type. |
| `SYS-TOK-006` | External linkage / interoperability. |
| `SYS-TOK-007` | Exchangeability. |
| `SYS-CASH-001` | A cash token is an instance of the definable token (SYS-TOK-005) whose unit is a money unit — any fiat currency, a CBDC, or a token-based coin/stablecoin on… |
| `SYS-CASH-002` | Every cash mint, transfer, and redemption MUST be journalled as a third entry (Section 5) so settlement is part of the shared triple-entry record. |
| `SYS-GOODS-001` | A goods token MUST represent a consignment/item, with the detailed record (description, quantity, provenance, custody history) off-chain (forked Postgres and… |
| `SYS-EDI-001` | Each commercial-document lifecycle MUST be a deterministic finite automaton whose states are UTXOs; a transition spends the current state UTXO and creates th… |
| `SYS-EDI-002` | The system MUST implement, as DFAs, the complete commercial-and-logistics document set, each with an explicit state set, event alphabet, and transition table… |
| `SYS-EDI-003` | Every document state transition MUST be journalled as a third entry (Section 5) and MUST be discoverable on chain by the US20220253835A1 method re-expressed… |
| `SYS-EDI-004` | Documents MUST cross-reference by object_id: an invoice references its PO and its consignment(s); a payment note references its invoice; a POD references its… |
| `SYS-EDI-005` | Decision (SYS-DECIDE-005, locked): the standards bridge IS provided, as a configurable, optional component enabled per trading partner. |
| `SYS-EDI-006` | The bridge MUST be a pure translation layer: inbound, it parses a standard message, validates it, and drives the corresponding DFA transition (journalled as… |
| `SYS-DOC-001` | Any commercial or logistics document (Section 9.2) — and the bill of lading in particular — MUST be renderable, on demand, to a complete human-readable PDF p… |
| `SYS-DOC-002` | The PDF MUST embed its own verifiability: the document's object_id, the current-state transaction reference as a BURI (SYS-PROOF-003), and the Merkle/SPV pro… |
| `SYS-DOC-003` | Rendering MUST be deterministic and reproducible: the same document at the same state MUST produce a byte-stable PDF (fixed templates, fixed field order, no… |
| `SYS-DOC-004` | Single-original / negotiability honesty. |
| `SYS-DOC-005` | The renderer MUST be invocable from the SQL surface (e.g. |
| `SYS-LOG-001` | A consignment MUST be a goods token (Section 8.2) whose lifecycle is a DFA per Section 9. |
| `SYS-LOG-002` | A goods record (description, HS/commodity code, quantity, weight, provenance, custody history, condition data, attached documents) MUST be stored in a separa… |
| `SYS-LOG-003` | The on-chain consignment metadata MUST carry a hash H2 representative of the goods-record details and the owner/custodian public key (US11210372). |
| `SYS-LOG-004` | Ownership/custody verification MUST follow US11210372: derive the parties' second public keys from their public keys and a generator value GV, compute the co… |
| `SYS-LOG-005` | Because verification reduces to a key match plus an H2→DHT lookup, any authorised party MUST be able to verify current ownership/custody of a consignment fro… |
| `SYS-LOG-006` | Where the consignment is governed by a negotiable bill of lading, the goods token MUST be the electronic B/L: transferring the token (a UTXO-lineage transfer… |
| `SYS-LOG-007` | The custody chain (shipper → carrier(s) → freight forwarder → customs → consignee) MUST be representable as successive CUSTODY_TRANSFER transitions, each re-… |
| `SYS-LOG-008` | Delivery acceptance (ACCEPTED) MUST be linkable to release of a cash token settlement (Section 8.1) and to the invoice/payment-note documents (Section 9), so… |
| `SYS-LOG-009` | Every off-chain record anchored on chain — a goods record (Section 10.2), an EDI/logistics document body (Section 9), or any DHT-stored asset — MUST be integ… |
| `SYS-LOG-010` | Records MUST be structured as header + body, where the header carries a hash of the body (and may carry the DHT key H2), so that the on-chain anchor commits… |
| `SYS-LOG-011` | Integrity (SYS-LOG-009, hash match) and ownership/custody (Section 10.2, controller-key vs DHT-registered-key match) are distinct, complementary checks and M… |
| `SYS-LOG-012` | The integrity check MUST back the PDF paper copy (Section 9.6): a rendered document's embedded references let a verifier recompute H3 over the rendered body… |
| `SYS-PROOF-001` | Any party (counterparty, auditor, regulator) MUST be able to prove that a given third-entry transaction — and therefore a recorded SQL change and its ECDH-HM… |
| `SYS-PROOF-002` | A Merkle-proof service entity MUST be providable that stores transaction identifiers and per-block subsets with their block headers (not the full chain, not… |
| `SYS-PROOF-003` | Inter-transaction references — the hash-chain back-reference of SYS-HMAC-008 and the cross-document references of SYS-EDI-004 — MUST be expressible as a BURI… |
| `SYS-PROOF-004` | For compact, privacy-preserving proof publication, the system MAY publish on chain the data items used to build a Merkle proof, split the proof into indexed… |
| `SYS-PROOF-005` | te_verify (SYS-PG-006) MUST export, for any row/change, a self-contained SPV proof (BURI + Merkle proof) verifiable by a third party with only block headers… |
| `SYS-CUST-001` | Cash tokens, goods tokens, and ledger master keys MUST be protectable by N-of-M threshold control in which the private key is never reconstructed: a threshol… |
| `SYS-CUST-002` | Key shares MUST be distributable and recoverable by the loss-resistant method of EP3259724B1: split the key (or a key-access element) into shares; transmit e… |
| `SYS-CUST-003` | Threshold authorisation MUST be expressed natively (bare OP_CHECKMULTISIG, or a single OP_CHECKSIG over a threshold-signed key — no P2SH). |
| `SYS-OVL-001` | The hash-chain streams (Section 5.4), document graphs (Section 9), and token lineages (Section 8) MUST be modelled as an overlay network on BSV data-storage… |
| `SYS-OVL-002` | Overlay keys MUST be derivable by a child-key-derivation (CKD) hierarchy whose key structure mirrors the overlay graph (EP4046048B1, claim 1 — a hierarchical… |
| `SYS-OVL-003` | Discovery and routing of an entity (stream, document, token, counterparty) MUST use its overlay address (graph position + derived key), with BURIs (SYS-PROOF… |
| `SYS-COMP-001` | For computations beyond a DFA transition — adjudicating a disputed logistics condition, computing a customs valuation, evaluating an oracle-fed predicate — t… |
| `SYS-COMP-002` | A resolved computation result MUST be feedable as an input event into the relevant DFA (Section 9), so a disputed or computed outcome advances the document/c… |
| `SYS-TEST-001` | All tests MUST run end-to-end against the Dockerised full BSV node on regtest before testnet/mainnet (SYS-CON-006). |
| `SYS-TEST-002` | The suite MUST include: ECDH-HMAC known-answer vectors (derive CS, K_hmac, tag from fixed inputs); hash-chain integrity tests (tamper → tag mismatch; drop/re… |
| `SYS-TEST-003` | Cross-implementation vectors MUST be shared between the C (fork) and TypeScript/Go components for ECDH, HKDF, HMAC, commitment, and encoding; divergence is a… |
| `SYS-REPO-001` | Monorepo (illustrative): |
| `SYS-PHASE-001` | Phased, each gated on its exit criteria: |
| `SYS-PHASE-002` | No phase begins before the prior phase's exit criteria pass. |
| `SYS-DECIDE-001` | Cash / token model: Resolved — a general definable token (SYS-TOK-005) for any item or money unit; all three cash profiles provided and selectable per token… |
| `SYS-DECIDE-002` | Write-interception mechanism: WAL logical decoding vs C extension vs in-core hooks. |
| `SYS-DECIDE-003` | Journalling mode: async outbox (default) vs synchronous commit-on-chain. |
| `SYS-DECIDE-004` | Confirmation depth for accounting finality, per use case. |
| `SYS-DECIDE-005` | EDI standards bridge: Resolved — provided as a configurable, optional, per-partner component (X12 850/855/856/810/820/214/990/210; EDIFACT ORDERS/ORDRSP/DESA… |
| `SYS-DECIDE-006` | Stream granularity: per-ledger vs per-account vs per-table hash chains. |
| `SYS-DECIDE-007` | Counterparty/auditor key model for single-party books (who is the ECDH counterparty when there is no trading partner). |
| `SYS-DECIDE-008` | PostgreSQL version to fork. |
| `SYS-DECIDE-010` | BSV node: Resolved — Teranode (via teranode-quickstart Docker; regtest for dev/test), SYS-NODE-001. |
| `SYS-DECIDE-009` | Controlled single-paper-original workflow for B/L: offered (with a PAPER_ISSUED token-locking state, SYS-DOC-004) or not. |
| `SYS-FINAL-001` | This is the contract for the build. |

---

## Appendix B — Definition of Done (acceptance criteria)

The build is **done** when every item below passes on the Teranode regtest stack (`SYS-NODE-001`), with cross-implementation vectors green (`SYS-TEST-003`):

1. **Crypto core.** ECDH common-secret (EP3860037A1 method), HKDF, HMAC-SHA256, commitments, and the canonical encoder/decoder pass known-answer vectors identically in the C fork and the TS/Go components.
2. **Keystone.** For a stream of SQL changes, each change produces `M(c)` to `GV` to `CS` to `K_hmac` to `tag(c)`, the tag is carried in a **spendable** locking script (no OP_RETURN), each tx carries `prev_txid`, and the stream forms a verifiable hash chain (`SYS-HMAC-001..011`).
3. **SQL surface.** Ordinary SQL writes, with no user blockchain code, produce correct on-chain third entries; `te_verify()` confirms tag + chain position; `te_render_pdf()` emits a deterministic, SPV-verifiable PDF (`SYS-PG-005/006`, `SYS-DOC-001..005`).
4. **Cold rebuild.** The entire database is reconstructed from the chain + master keys alone and asserts byte-equality with the live database (`SYS-PG-004`).
5. **No-P2SH / no-OP_RETURN.** A static check over all produced scripts confirms zero P2SH patterns and zero OP_RETURN outputs; all data rides in spendable scripts (`SYS-CON-002`, `SYS-CON-008`, `SYS-ENC-001`).
6. **Sighash.** All state-transition signatures set `SIGHASH_FORKID` and bind successor outputs (`SYS-ENC-004`).
7. **Definable token.** A new token type is defined via the schema with no code change; cash (issuer-backed / satoshi-tagged / pegged), a CBDC/stablecoin-linked token (via adapter contract), and a goods token all mint/transfer/redeem and journal as third entries; any two instances swap atomically (`SYS-TOK-005..007`, `SYS-CASH-001`).
8. **EDI + bridge.** Every document type in `SYS-EDI-002` runs its DFA lifecycle; the optional X12/EDIFACT bridge translates the listed message types in and out and can be disabled without affecting the core (`SYS-EDI-005/006`).
9. **Logistics.** Consignment lifecycle, multi-party custody, bill-of-lading-as-token title transfer, ownership (key-match, US11210372) and integrity (hash-match, GB2558485A) checks, and delivery-versus-payment all pass (`SYS-LOG-001..012`).
10. **Proofs / custody / overlay / computation.** SPV inclusion proof + BURI export; threshold custody (sign without reconstructing the key; loss-resistant share storage); overlay addressing via CKD hierarchy; staked-computation resolution feeding a DFA event (`SYS-PROOF-*`, `SYS-CUST-*`, `SYS-OVL-*`, `SYS-COMP-*`).
11. **Resilience.** Reorg handling, outbox idempotency across restarts, and confirmation-depth gating behave per `SYS-PG-007`.
12. **Integrity of grounding.** No requirement cites a patent abstract; every patent claim referenced is from text read in full (`SYS-INTEG-001`).

A failure of any single item blocks release (consistent with the review standard: a minor failure is sufficient for rejection).

---

## Appendix C — Glossary

- **Third entry** — the on-chain, ECDH-HMAC-bound, hash-chained record of a SQL change; the shared receipt both parties reconcile against.
- **CS (common secret)** — secp256k1 ECDH secret derived per change from deterministic sub-keys; the HMAC-key input (EP3860037A1).
- **GV (generator value)** — `SHA-256(M(c))`; the scalar that derives the per-change sub-keys.
- **M(c)** — canonical message identifying a SQL change (table, row, column, op, seq, prev_txid); public.
- **tag(c)** — `HMAC-SHA256(HKDF(CS, ...), change_image)`; carried in a spendable locking script.
- **Hash chain / stream** — the linked sequence of third-entry transactions (each carrying `prev_txid`).
- **DFA** — deterministic finite automaton; a document/consignment lifecycle whose states are UTXOs (US20220253835A1).
- **DHT** — distributed hash table; off-chain store for goods records and document bodies, keyed by `H2`.
- **H1/H2/H3/H4** — asset body hash / DHT-key hash of details / freshly computed integrity hash / DHT-stored integrity hash (US11210372, GB2558485A).
- **BURI** — Blockchain Uniform Resource Indicator; delimiter-separated block-id/tx-id/Merkle-proof string, SPV-verifiable without the block payload (WO2022214264A1).
- **Definable token** — the universal token primitive (`SYS-TOK-005`): any item or money unit, issuer-defined; cash, CBDC-linked, stablecoin-linked, and goods are instances (EP3748903A1).
- **CTO** — the Confidential Transferable Object substrate primitive (separate spec) on which this system is built; Tiers F/S/T are its deletion/custody profiles.
- **Teranode** — the chosen BSV node (Go microservices; teranode-quickstart Docker; regtest for dev/test).
- **FORKID / OTDA** — `SIGHASH_FORKID` (always set on BSV); Original Transaction Digest Algorithm (Chronicle opt-in via the `0x20` sighash bit).

*End of Release 8.0.*
