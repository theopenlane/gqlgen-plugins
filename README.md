[![Go Report Card](https://goreportcard.com/badge/github.com/theopenlane/gqlgen-plugins)](https://goreportcard.com/report/github.com/theopenlane/gqlgen-plugins)
[![Build status](https://badge.buildkite.com/651bd9d2d92df64fcab6bab5db1842565d29c48ade77b52bd7.svg)](https://buildkite.com/theopenlane/gqlgen-plugins?branch=main)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=theopenlane_gqlgen-plugins&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=theopenlane_gqlgen-plugins)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)

# gqlgen-plugins

[gqlgen](https://gqlgen.com/reference/plugins/) provides a way to hook into the
gqlgen code generation lifecycle. This repo contains several hooks that can be
used:

- bulkgen
- resovlergen
- searchgen
- fieldgen

## ResolverGen

This hook will override the default generated resolver functions with the
templates for CRUD operations.

## BulkGen

Creates resolvers to do bulk operations for a schema for both bulk input or a
csv file upload input.

## FieldGen

This plugin is designed to programmatically add additional fields to your graphql schema based on existing fields
in the schema or the schema name

## SearchGen

Creates search resolvers to search on fields within the ent schema. You must
pass in the package import name of the `generated` ent code, e.g.
`github.com/theopenlane/core/internal/ent/generated`. If the package is not
named `generated` it is added as an alias.

```go
api.AddPlugin(searchgen.New("github.com/theopenlane/core/internal/ent/generated")), // add the search plugin
```

## Usage

Add the plugins to the `generate.go` `main` function to be included in the
setup:

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
		api.AddPlugin(searchgen.New("github.com/theopenlane/core/internal/ent/generated")), // add the search plugin
	); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
```
