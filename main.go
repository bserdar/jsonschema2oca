package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/santhosh-tekuri/jsonschema"
)

var idGen int

type Index struct {
	names []string
	ids   map[string]string
}

func (i *Index) add(name string) string {
	id := idGen
	idGen++
	i.names = append(i.names, name)
	if i.ids == nil {
		i.ids = make(map[string]string)
	}
	idstr := fmt.Sprintf("id_%s_%d", name, id)
	i.ids[idstr] = name
	return idstr
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("json2oca <entity mappings file>")
		return
	}
	var objects map[string]string
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, &objects); err != nil {
		panic(err)
	}
	compiler := jsonschema.NewCompiler()
	schemas := make(map[string]*jsonschema.Schema)
	schemaNames := make(map[*jsonschema.Schema]string)
	for name, ref := range objects {
		sch, err := compiler.Compile(ref)
		if err != nil {
			panic(err)
		}
		schemas[name] = sch
		schemaNames[sch] = name
	}
	for name, sch := range schemas {
		index := Index{}
		s, err := decompose(name, sch, schemaNames, &index)
		if err != nil {
			panic(err)
		}

		writeJSON(name, "base", s.toBaseSchema())

		types := map[string]string{}
		s.getTypes(types)
		writeJSON(name, "type", map[string]interface{}{"@context": "https://oca.tech/v1",
			"base":  name,
			"types": types})

		formats := map[string]string{}
		s.getFormats(formats)
		writeJSON(name, "format", map[string]interface{}{"@context": "https://oca.tech/v1",
			"base":    name,
			"formats": formats})

		writeJSON(name, "index", map[string]interface{}{"@context": "https://oca.tech/v1",
			"base":  name,
			"index": index.ids})
	}
}

func writeJSON(name, typ string, data interface{}) {
	x, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(fmt.Sprintf("%s_%s.json", name, typ), x, 0660)
}

type ArraySchema struct {
	Items SchemaProperty `json:"items"`
}

func (a ArraySchema) toBaseSchema() interface{} {
	return a.Items.toBaseSchema()
}

func (a ArraySchema) getTypes(id string, types map[string]string) {
	a.Items.getTypes(id, types)
}

func (a ArraySchema) getFormats(id string, formats map[string]string) {
	a.Items.getFormats(id, formats)
}

type ObjectSchema struct {
	Fields map[string]SchemaProperty
}

func (o *ObjectSchema) Set(key string, value SchemaProperty, indexes *Index) {
	if o.Fields == nil {
		o.Fields = make(map[string]SchemaProperty)
	}
	ID := indexes.add(key)
	o.Fields[ID] = value
}

func (o ObjectSchema) toBaseSchema() interface{} {
	ret := map[string]interface{}{}
	for k, v := range o.Fields {
		ret[k] = v.toBaseSchema()
	}
	return ret
}

func (o ObjectSchema) getTypes(id string, types map[string]string) {
	for k, v := range o.Fields {
		v.getTypes(k, types)
	}
}

func (o ObjectSchema) getFormats(id string, formats map[string]string) {
	for _, v := range o.Fields {
		v.getFormats(id, formats)
	}
}

type SchemaProperty struct {
	Reference string           `json:"reference,omitempty"`
	Object    *ObjectSchema    `json:"object,omitempty"`
	Array     *ArraySchema     `json:"array,omitempty"`
	OneOf     []SchemaProperty `json:"oneOf,omitempty"`
	Type      string
	Format    string
}

func (p SchemaProperty) toBaseSchema() interface{} {
	if len(p.Reference) > 0 {
		return map[string]interface{}{"reference": p.Reference}
	}
	if p.Object != nil {
		return map[string]interface{}{"object": p.Object.toBaseSchema()}
	}
	if p.Array != nil {
		return map[string]interface{}{"array": p.Array.toBaseSchema()}
	}
	if len(p.OneOf) != 0 {
		arr := make([]interface{}, 0)
		for _, c := range p.OneOf {
			arr = append(arr, c.toBaseSchema())
		}
		return map[string]interface{}{"oneOf": arr}
	}
	return "value"
}

func (p SchemaProperty) getTypes(id string, types map[string]string) {
	if p.Object != nil {
		p.Object.getTypes(id, types)
		return
	}
	if p.Array != nil {
		p.Array.getTypes(id, types)
	}
	if p.OneOf != nil {
		for _, x := range p.OneOf {
			x.getTypes(id, types)
		}
	}
	if len(p.Type) > 0 {
		types[id] = p.Type
	}
}

func (p SchemaProperty) getFormats(id string, formats map[string]string) {
	if p.Object != nil {
		p.Object.getFormats(id, formats)
		return
	}
	if p.Array != nil {
		p.Array.getFormats(id, formats)
	}
	if p.OneOf != nil {
		for _, x := range p.OneOf {
			x.getFormats(id, formats)
		}
	}
	if len(p.Format) > 0 {
		formats[id] = p.Format
	}
}

type Schema struct {
	Name       string       `json:"name"`
	Attributes ObjectSchema `json:"attributes"`
}

func (s Schema) toBaseSchema() interface{} {
	ret := map[string]interface{}{
		"@context":   "https://oca.tech/v1",
		"name":       s.Name,
		"attributes": s.Attributes.toBaseSchema()}
	return ret
}

func (s Schema) getTypes(types map[string]string) {
	s.Attributes.getTypes("", types)
}

func (s Schema) getFormats(formats map[string]string) {
	s.Attributes.getFormats("", formats)
}

// TODO: propertyNames
// patternProperties
// additionalItems

func decompose(objectName string, objectSchema *jsonschema.Schema, nameMap map[*jsonschema.Schema]string, indexes *Index) (Schema, error) {
	ret := Schema{}
	ret.Name = objectName

	s := SchemaProperty{}
	loop := make([]*jsonschema.Schema, 0)
	if err := decomposeSchema(&s, objectSchema, nameMap, indexes, loop); err != nil {
		return ret, err
	}
	if s.Object == nil {
		return ret, fmt.Errorf("%s base schema is not an object", objectName)
	}
	ret.Attributes = *s.Object

	return ret, nil
}

func decomposeSchema(target *SchemaProperty, sch *jsonschema.Schema, nameMap map[*jsonschema.Schema]string, indexes *Index, loop []*jsonschema.Schema) error {
	for _, x := range loop {
		if sch == x {
			return fmt.Errorf("Loop: %+v\n +: %+v", loop, sch)
		}
	}
	loop = append(loop, sch)

	switch {
	case sch.Ref != nil:
		ref, ok := nameMap[sch.Ref]
		if ok {
			target.Reference = ref
			return nil
		}
		return decomposeSchema(target, sch.Ref, nameMap, indexes, loop)

	case len(sch.AllOf) > 0:
		panic("allOf not handled")

	case len(sch.AnyOf) > 0:
		panic("anyOf not handled")

	case len(sch.OneOf) > 0:
		for _, x := range sch.OneOf {
			prop := SchemaProperty{}
			if err := decomposeSchema(&prop, x, nameMap, indexes, loop); err != nil {
				return err
			}
			target.OneOf = append(target.OneOf, prop)
		}

	case len(sch.Properties) > 0:
		target.Object = &ObjectSchema{}
		for k, v := range sch.Properties {
			val := SchemaProperty{}
			if err := decomposeSchema(&val, v, nameMap, indexes, loop); err != nil {
				return err
			}
			target.Object.Set(k, val, indexes)
		}
		// TODO: patternProperties, etc
	case sch.Items != nil:
		target.Array = &ArraySchema{}
		if itemSchema, ok := sch.Items.(*jsonschema.Schema); ok {
			if err := decomposeSchema(&target.Array.Items, itemSchema, nameMap, indexes, loop); err != nil {
				return err
			}
		} else {
			panic("Multiple item schemas not supported")
		}
		// TODO: additionalItems, etc
	default:
		if len(sch.Types) > 0 {
			target.Type = sch.Types[0]
		}
		target.Format = sch.FormatName
	}

	return nil
}
