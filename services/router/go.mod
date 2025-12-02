module github.com/vsc-eco/vsc-dex-mapping/services/router

go 1.24.0

require (
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.11.1
	github.com/vsc-eco/vsc-dex-mapping/schemas v0.0.0
	github.com/vsc-eco/vsc-dex-mapping/sdk/go v0.0.0
	github.com/xeipuuv/gojsonschema v1.2.0
)

replace github.com/vsc-eco/vsc-dex-mapping/schemas => ../../schemas
replace github.com/vsc-eco/vsc-dex-mapping/sdk/go => ../../sdk/go
replace vsc-node => ../../../go-vsc-node

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hasura/go-graphql-client v0.12.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.11 // indirect
)
