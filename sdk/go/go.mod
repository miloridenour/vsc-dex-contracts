module github.com/vsc-eco/vsc-dex-mapping/sdk/go

go 1.24.0

require (
	github.com/hasura/go-graphql-client v0.12.2
	golang.org/x/crypto v0.21.0
	vsc-node v0.0.0
)

replace vsc-node => ../../../go-vsc-node
