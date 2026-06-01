module te-bsv/services-go

go 1.26.3

require (
	github.com/bsv-blockchain/go-sdk v1.2.24
	te-bsv/cryptocore v0.0.0
)

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.52.0 // indirect
)

replace te-bsv/cryptocore => ../crypto-core/go
