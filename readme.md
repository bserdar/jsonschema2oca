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

  jsonschema2oca <entity mappings file>
  
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

In order to map the structure of entities into OCA, I had to make
several assumptions about the OCA base schema that are not available
as of this writing:

  * Schema layout:
```
{
  @context: https://oca.tech/v1
  name: <base schema name>
  attributes: {
     <attribute definition>,
     ...
  }
}
```
  * Simple attributes are defined as:
```
    "key": "value"
```
  * Nested objects are defined as:
```
    "key": {
      "object" {
         <attribute definition>...
      }
    }
```
  * Arrays are defined as:
```
    "key": {
      "array": { <array item definition> }
    }
```
    where array item definition can be a reference, object, array, or "value"

  * References are defined as:
```
    "key": {
       "reference": <schema name>
    }
```
  * oneOf constructs are defined as:
```
   "key": {
      "oneOf": [
         <item definition>
         ...
      ]
    }
```
