# Experimental JSON schema to OCA layers translator

This tool compiles a JSON schema and decomposes it to OCA-compliant
layers. These layers are:

  * Base schema, containing normalized field names and object
    structure,
  * An index overlay that assigns names to fields defined in the base
    schema,
  * A type overlay that assigns JSON types to the fields of the based
    schema,
  * A format overlay that contains the JSON format directives for the
    fields.
    
## Building

Use the go build system. Clone the repository and run:

  go build 

## Running

After building:

```
  jsonschema2oca <entity mappings file> <target prefix>
``` 
  
The entity mappings file is a JSON file of the form:

```
{
  "EntityName" : {
    "reference": "schema reference",
    "blinding": [
       field names,...
    ]
  },
  ...
}
```

The program will write OCA layers for each entity.

For the FHIR schema this is defined as:

```
{
      "Account": {
        "reference": "testdata/fhir.schema.json#/definitions/Account",
        "blinding": []
       },
      ...
}
```

The program builds the schema for each of these entities using their
schema references, decomposes each entity into its base + overlays,
and writes the resulting files. 

At this point, the program does not support allOf, anyOf, if,
patternProperties, additionalItems constructs of the JSON schema.

## Cyclic references

The program recursively descends the compiled schema. If it detects
any cycles, it will print this out. You have to break all cycles by
defining additional entries in the entity mappings so that cycles can
be rewritten as references.

For instance, if the schema objects are defined as:
```
 A -> B -> C -> A
```

and entities as:
```
{
  "EntityB" : "#/B",
}
```

then the unfolding of B is unbounded. So define another entity to
break the loop:

```
{
  "EntityB": "#/B",
  "EntityA": "#/A"
}
```

With this input, the unfolding of B will contain all elements of C,
but will have a reference to A.


## OCA Compliance

The generated OCA schemas are compliant to 

https://github.com/bserdar/oca-spec/tree/format/RFCs/003-bserdar-schema-format
