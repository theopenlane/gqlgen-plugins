# FieldGen

A graphql plugin to conditionally add fields to schema(s) which are not part of the existing model.

## Overview

This plugin is for users of the `ent` project who use `ent` in conjunction with GraphQL. When you add additional fields with `ent` using their `model/fields/additional`, ex:

```
{{ define "model/fields/additional" }}
        // CreatedByMeow includes the meow details about the user or service that created the object
        CreatedByMeow string `json:"createdByMeow,omitempty"`
        // UpdatedByMeow includes the meow details about the user or service that last updated the object
        UpdatedByMeow string `json:"updatedByMeow,omitempty"`
    {{- end }}
{{ end }}
```

You can manually extend your graphql schema, ex:

```graphql
extend type Meowzers {
    CreatedByMeow: String
    UpdatedByMeow: String
}
```

However, manually managing these fields with many scehmas (and ensuring future schemas are extended similarly) becomes difficult and unwieldy, which is where the `fieldgen` plugin comes into play.

## Usage

Once you've added your additional fields (in whatever manner you want) you can use this plugin to add those fields to your graphql schema.

In the below example, we want a new field on every existing schema that has the `createdByID` and `updatedByID` field:

```go
var extraFields = []fieldgen.AdditionalField{
	{
		Name:                         "createdByMeow",
		Type:                         "String",
		NonNull:                      true,
		Description:                  "The cat who created the object",
		AddToSchemaWithExistingField: "createdByID",
	},
	{
		Name:                         "updatedByMeow",
		Type:                         "String",
		NonNull:                      true,
		Description:                  "The cat who last updated the object",
		AddToSchemaWithExistingField: "updatedByID",
	},
}
```

Add the plugin to your `generate` function:

```go
	if err := api.Generate(cfg,
		api.AddPlugin(fieldgen.NewExtraFieldsGen(extraFields)), // add the fieldgen plugin
	); err != nil {
		log.Fatal().Err(err).Msg("failed to generate gqlgen server")
	}
```

This uses the `MutateConfig` plugin option so the fields are added before the code is generated.

After running `generate`, you should see the additional fields in your model and graphql schema.

## CustomTypes

If you are using a custom type, you can either manually add the scalar to your schema or you can use the `CustomType` field instead of `Type`. This will automatically add the type to the respective sources.