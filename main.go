package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/santhosh-tekuri/jsonschema"
)

var idGen int
var targetNS string

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

type Entity struct {
	Name      string
	Reference string   `json:"reference"`
	Blinding  []string `json:"blinding"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("json2oca <entity mappings file> <targetNS>")
		return
	}
	targetNS = os.Args[2]
	var objects map[string]Entity
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, &objects); err != nil {
		panic(err)
	}
	for k, v := range objects {
		v.Name = k
		objects[k] = v
	}
	compiler := jsonschema.NewCompiler()
	schemas := make(map[string]*jsonschema.Schema)
	schemaNames := make(map[*jsonschema.Schema]string)
	for name, e := range objects {
		sch, err := compiler.Compile(e.Reference)
		if err != nil {
			panic(err)
		}
		schemas[name] = sch
		schemaNames[sch] = name
	}
	for name, sch := range schemas {
		s, err := decompose(name, sch, schemaNames)
		if err != nil {
			panic(err)
		}

		writeJSON(name, "base", s.toBaseSchema())

		types := map[string]string{}
		s.getTypes(types)
		t := []map[string]interface{}{}
		for k, v := range types {
			t = append(t, map[string]interface{}{"key": k, "type": v})
		}
		writeJSON(name, "type", map[string]interface{}{"@context": []string{"http://schemas.cloudprivacylabs.com/Overlay", "http://schemas.cloudprivacylabs.com/TypeOverlay"},
			"base":      targetNS + name,
			"selectors": t})

		formats := map[string]string{}
		t = []map[string]interface{}{}
		for k, v := range formats {
			t = append(t, map[string]interface{}{"key": k, "format": v})
		}
		s.getFormats(formats)
		writeJSON(name, "format", map[string]interface{}{"@context": []string{"http://schemas.cloudprivacylabs.com/Overlay", "http://schemas.cloudprivacylabs.com/FormatOverlay"},
			"base":      targetNS + name,
			"selectors": t})

		writeJSON(name, "index", s.toIndex())
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

func (a ArraySchema) toIndex() interface{} {
	x := a.Items.toIndex()
	if x == nil {
		return nil
	}
	return x
}

func (a ArraySchema) getTypes(name string, types map[string]string) {
	a.Items.getTypes(name, types)
}

func (a ArraySchema) getFormats(name string, formats map[string]string) {
	a.Items.getFormats(name, formats)
}

type ObjectSchema struct {
	Fields map[string]SchemaProperty
}

func (o *ObjectSchema) Set(key string, value SchemaProperty) {
	if o.Fields == nil {
		o.Fields = make(map[string]SchemaProperty)
	}
	o.Fields[key] = value
}

func (o ObjectSchema) toBaseSchema() interface{} {
	ret := []map[string]interface{}{}
	for k, v := range o.Fields {
		m := map[string]interface{}{"key": "_id_" + k}
		for x, y := range v.toBaseSchema() {
			m[x] = y
		}
		ret = append(ret, m)
	}
	return ret
}

func (o ObjectSchema) toIndex() interface{} {
	var ret []map[string]interface{}
	for k, v := range o.Fields {
		if ret == nil {
			ret = []map[string]interface{}{}
		}
		m := map[string]interface{}{"key": "_id_" + k, "name": k}

		ix := v.toIndex()
		if ix != nil {
			for x, y := range ix {
				if y != nil {
					m[x] = y
				}
			}
		}
		ret = append(ret, m)
	}
	if ret == nil {
		return nil
	}
	return ret
}

func (o ObjectSchema) getTypes(name string, types map[string]string) {
	for k, v := range o.Fields {
		var newName string
		if len(name) == 0 {
			newName = k
		} else {
			newName = name + "." + k
		}
		v.getTypes(newName, types)
	}
}

func (o ObjectSchema) getFormats(name string, formats map[string]string) {
	for k, v := range o.Fields {
		var newName string
		if len(name) == 0 {
			newName = k
		} else {
			newName = name + "." + k
		}
		v.getFormats(newName, formats)
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

func (p SchemaProperty) toBaseSchema() map[string]interface{} {
	if len(p.Reference) > 0 {
		return map[string]interface{}{"reference": targetNS + p.Reference}
	}
	if p.Object != nil {
		return map[string]interface{}{"attributes": p.Object.toBaseSchema()}
	}
	if p.Array != nil {
		return map[string]interface{}{"items": p.Array.toBaseSchema()}
	}
	if len(p.OneOf) != 0 {
		arr := make([]interface{}, 0)
		for _, c := range p.OneOf {
			arr = append(arr, c.toBaseSchema())
		}
		return map[string]interface{}{"oneOf": arr}
	}
	return map[string]interface{}{}
}

func (p SchemaProperty) toIndex() map[string]interface{} {
	if len(p.Reference) > 0 {
		return nil
	}
	if p.Object != nil {
		x := p.Object.toIndex()
		if x != nil {
			return map[string]interface{}{"attributes": x}
		}
		return nil
	}
	if p.Array != nil {
		x := p.Array.toIndex()
		if x != nil {
			return map[string]interface{}{"items": x}
		}
		return nil
	}
	if len(p.OneOf) != 0 {
		arr := make([]interface{}, 0)
		for _, c := range p.OneOf {
			x := c.toIndex()
			if x != nil {
				arr = append(arr, c.toIndex())
			}
		}
		if len(arr) == 0 {
			return nil
		}
		return map[string]interface{}{"oneOf": arr}
	}
	return nil
}

func (p SchemaProperty) getTypes(name string, types map[string]string) {
	if p.Object != nil {
		p.Object.getTypes(name, types)
		return
	}
	if p.Array != nil {
		p.Array.getTypes(name, types)
	}
	if p.OneOf != nil {
		for _, x := range p.OneOf {
			x.getTypes(name, types)
		}
	}
	if len(p.Type) > 0 {
		types[name] = p.Type
	}
}

func (p SchemaProperty) getFormats(name string, formats map[string]string) {
	if p.Object != nil {
		p.Object.getFormats(name, formats)
		return
	}
	if p.Array != nil {
		p.Array.getFormats(name, formats)
	}
	if p.OneOf != nil {
		for _, x := range p.OneOf {
			x.getFormats(name, formats)
		}
	}
	if len(p.Format) > 0 {
		formats[name] = p.Format
	}
}

type Schema struct {
	Name       string       `json:"name"`
	Attributes ObjectSchema `json:"attributes"`
}

func (s Schema) toBaseSchema() interface{} {
	ret := map[string]interface{}{
		"@context":   "http://schemas.cloudprivacylabs.com/BaseSchema",
		"@id":        targetNS + s.Name,
		"attributes": s.Attributes.toBaseSchema()}
	return ret
}

func (s Schema) toIndex() interface{} {
	ret := map[string]interface{}{
		"@context": []string{"http://schemas.cloudprivacylabs.com/Overlay",
			"http://schemas.cloudprivacylabs.com/IndexOverlay"},
		"base":       targetNS + s.Name,
		"attributes": s.Attributes.toIndex()}
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

func decompose(objectName string, objectSchema *jsonschema.Schema, nameMap map[*jsonschema.Schema]string) (Schema, error) {
	ret := Schema{}
	ret.Name = objectName

	s := SchemaProperty{}
	loop := make([]*jsonschema.Schema, 0)
	if err := decomposeSchema(&s, objectSchema, nameMap, loop); err != nil {
		return ret, err
	}
	if s.Object == nil {
		return ret, fmt.Errorf("%s base schema is not an object", objectName)
	}
	ret.Attributes = *s.Object

	return ret, nil
}

func decomposeSchema(target *SchemaProperty, sch *jsonschema.Schema, nameMap map[*jsonschema.Schema]string, loop []*jsonschema.Schema) error {
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
		return decomposeSchema(target, sch.Ref, nameMap, loop)

	case len(sch.AllOf) > 0:
		panic("allOf not handled")

	case len(sch.AnyOf) > 0:
		panic("anyOf not handled")

	case len(sch.OneOf) > 0:
		for _, x := range sch.OneOf {
			prop := SchemaProperty{}
			if err := decomposeSchema(&prop, x, nameMap, loop); err != nil {
				return err
			}
			target.OneOf = append(target.OneOf, prop)
		}

	case len(sch.Properties) > 0:
		target.Object = &ObjectSchema{}
		for k, v := range sch.Properties {
			val := SchemaProperty{}
			if err := decomposeSchema(&val, v, nameMap, loop); err != nil {
				return err
			}
			target.Object.Set(k, val)
		}
		// TODO: patternProperties, etc
	case sch.Items != nil:
		target.Array = &ArraySchema{}
		if itemSchema, ok := sch.Items.(*jsonschema.Schema); ok {
			if err := decomposeSchema(&target.Array.Items, itemSchema, nameMap, loop); err != nil {
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
