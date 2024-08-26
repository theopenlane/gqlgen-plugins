# gqlgen-plugins

[gqlgen](https://gqlgen.com/reference/plugins/) provides a way to hook into the gqlgen code generation lifecycle. This repo contains two hooks:
- bulkgen
- resovlergen

## ResolverGen

This hook will override the default generated resolver functions with the templates for CRUD operations.

## BulkGen

Creates resolvers to do bulk operations for a schema for both bulk input or a csv file upload input.

## Usage

Add the plugins to the `generate.go` `main` function to be included in the setup:

```go
func main() {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}

	if err := api.Generate(cfg,
		api.ReplacePlugin(resolvergen.New()), // replace the resolvergen plugin
		api.AddPlugin(bulkgen.New()),         // add the bulkgen plugin
	); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
```

