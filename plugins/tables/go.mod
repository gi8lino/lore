module io.lore/tables

go 1.27.0

require (
	github.com/gi8lino/lore v0.0.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/net v0.59.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/gi8lino/lore => ../..
