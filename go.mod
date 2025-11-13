module github.com/vsc-eco/vsc-dex-mapping

go 1.24.0

require (
	github.com/stretchr/testify v1.9.0
	github.com/vsc-eco/vsc-dex-mapping/sdk/go v0.0.0
)

replace github.com/vsc-eco/vsc-dex-mapping/sdk/go => ./sdk/go
replace vsc-node => ../go-vsc-node

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vsc-eco/vsc-dex-mapping/sdk/go => ./sdk/go
