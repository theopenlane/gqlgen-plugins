# FieldGen

A graphql plugin to conditionally add additional fields to schema(s) that are not part
of the existing model.

## Details

So you are using ent, and you added some additional fields using their `model/fields/additional`:

```
{{ define "model/fields/additional" }}
        // CreatedByMeow includes the meow details about the user or service that created the object
        CreatedByMeow string `json:"createdByMeow,omitempty"`
        // UpdatedByMeow includes the meow details about the user or service that last updated the object
        UpdatedByMeow string `json:"updatedByMeow,omitempty"`
    {{- end }}
{{ end }}
```

Now in the simple case, you can manually extend your graphql schema using:

```graphql
extend type Meowzers {
    CreatedByMeow: String
	UpdatedByMeow: String
}
```

But what if you added it all your schemas, and not just one, and you want all future schemas to also include these fields. You could go manually add those, but computers do this better, so here comes the `fieldgen` plugin.


## Usage

Once you add your additional fields, in whatever manner you want, you can then use this plugin to add them to your graphql schema.

In this example, I want this  new field on every existing schema that has the `createdByID` and `updatedByID` field:

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

You can then add the plugin to your generate function:

```
	if err := api.Generate(cfg,
		api.AddPlugin(fieldgen.NewExtraFieldsGen(extraFields)), // add the fieldgen plugin
	); err != nil {
		log.Fatal().Err(err).Msg("failed to generate gqlgen server")
	}
```

This uses the `MutateConfig` plugin option so the fields are added before the code is generated.

Running generate, you should see the additional fields in your model and graphql schema.

## CustomTypes

If you are using a custom type, you can either manually add the scalar to your schema or
you can use the `CustomType` field instead of `Type`, this will automatically add the type
to the sources.