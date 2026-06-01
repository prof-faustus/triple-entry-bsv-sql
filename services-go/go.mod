module te-bsv/services-go

go 1.26.3

require (
	github.com/bsv-blockchain/go-sdk v1.2.24
	github.com/jackc/pgx/v5 v5.9.2
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	te-bsv/cryptocore v0.0.0
)

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace te-bsv/cryptocore => ../crypto-core/go
