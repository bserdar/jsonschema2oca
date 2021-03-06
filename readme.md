# Experimental JSON schema to OCA layers translator

This tool compiles a JSON schema and decomposes it to OCA-compliant
layers. These layers are:

  * Schema base, containing normalized field names and object
    structure,
  * An index overlay that assigns names to fields defined in the schema base,
  * A type overlay that assigns JSON types to the fields of the schema
    base,
  * A format overlay that contains the JSON format directives for the
    fields.
    
## OCA Info

> OCA is an architecture that presents a schema as a multi-dimensional object consisting of a stable schema base and interoperable overlays. 

OCA information: https://wiki.trustoverip.org/display/HOME/Semantic+Domain+Group

OCA is flat. It is primarily intended for data capture. However, OCA
may have many use cases in data exchange/data processing use-cases where
data is rarely flat. This RFC proposes changes to the OCA schema to
support nested and array-like structures, that is, JSON:

https://github.com/bserdar/oca-spec/tree/format/RFCs/003-bserdar-schema-format

So if goes from this:

![OCA](img/schema-base-and-overlays.png)

To this:

![OCA](img/schema-base-and-overlays-linked.png)
    
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
