# crypto-core/go — Go side of the shared crypto core

Module `te-bsv/cryptocore`. Byte-exact contract: [`../../spec/ALGORITHMS.md`](../../spec/ALGORITHMS.md).

Implements the keystone primitives (`SYS-HMAC-002/003/004/009`, `SYS-ENC-005`): canonical
encoder/decoder, `M(c)` + field-record encoding, GV/sub-key derivation and the EP3860037A1 ECDH common
secret, HKDF-SHA256, HMAC-SHA256, the SHA-256 blinded commitment, and AES-256-GCM (`SYS-SUB-001`).

Dependencies: `github.com/decred/dcrd/dcrec/secp256k1/v4` (secp256k1 scalar/point ops),
`golang.org/x/crypto/hkdf`; everything else is the Go standard library.

```
go test ./...      # consumes ../vectors/*.json; must match the TS impl byte-for-byte (SYS-TEST-003)
go vet ./...
```
