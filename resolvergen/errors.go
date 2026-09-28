package resolvergen

import "errors"

// ErrGraphResolverDirRequired is returned when UpdateWorkflowResolvers is called without a directory
var ErrGraphResolverDirRequired = errors.New("graphResolverDir is required")

// ErrModuleRootNotFound is returned when the module root cannot be determined from import paths or go.mod
var ErrModuleRootNotFound = errors.New("unable to determine module root")

// ErrGraphSchemaDirRequired is returned when WriteCatalogSchema is called without a directory
var ErrGraphSchemaDirRequired = errors.New("graphSchemaDir is required")

// ErrCatalogListQueryNotFound is returned when a catalog schema has no connection query to derive the catalog query from
var ErrCatalogListQueryNotFound = errors.New("catalog schema has no connection query")

// ErrCatalogPackagesRequired is returned when catalog schemas are set without the rule, entityops, and jsonx packages
var ErrCatalogPackagesRequired = errors.New("rule, entityops, and jsonx packages are required for catalog schemas")
