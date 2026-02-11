package visitors

import (
	"github.com/google/jsonschema-go/jsonschema"
)

type JSONSchemaVisitor struct {
	Schemas         map[string]*jsonschema.Schema
	required_fields []string
	current_element string
}

func NewJSONSchemaVisitor() *JSONSchemaVisitor {
	return &JSONSchemaVisitor{
		Schemas: map[string]*jsonschema.Schema{},
	}
}

// We only care about assignable relations here - all others are readonly and irrelvant for input validation

// For logical operations, coalesce to a non-nil body or nil
func (v *JSONSchemaVisitor) VisitAnd(left any, right any) any {
	if left != nil {
		return left
	}

	return right
}

func (v *JSONSchemaVisitor) VisitOr(left any, right any) any {
	if left != nil {
		return left
	}

	return right
}

func (v *JSONSchemaVisitor) VisitUnless(left any, right any) any {
	if left != nil {
		return left
	}

	return right
}

// Relation references are nil (and coalesce out above)
func (v *JSONSchemaVisitor) VisitRelationExpression(name string) any {
	return nil
}

func (v *JSONSchemaVisitor) VisitSubRelationExpression(name string, sub string) any {
	return nil
}

// Capture details about what's assignable
func (v *JSONSchemaVisitor) VisitAssignableExpression(typeNamespace string, typeName string, cardinality string, _data_type any) any {
	data_type := _data_type.(*jsonschema.Schema)
	switch cardinality {
	case "AtMostOne": //Optional, individual value
		return v.handleIndividualAssignable(false, data_type)
	case "ExactlyOne": //Required, individual value
		return v.handleIndividualAssignable(true, data_type)
	case "All": //Required, individual value. Type should look like: resource_type:*
		return v.handleIndividualAssignable(false, data_type)
	case "AtLeastOne": //Required, array
		return v.handleArrayAssignable(true, data_type)
	case "Any": //Optional, array
		return v.handleArrayAssignable(false, data_type)
	default:
		panic("Cardinality not matched: " + cardinality)
	}
}

func (v *JSONSchemaVisitor) handleIndividualAssignable(required bool, _data_type any) any {
	data_type := _data_type.(*jsonschema.Schema)
	if required {
		v.required_fields = append(v.required_fields, v.current_element)
	}

	return data_type
}

func (v *JSONSchemaVisitor) handleArrayAssignable(required bool, _data_type any) any {
	data_type := _data_type.(*jsonschema.Schema)

	arr := &jsonschema.Schema{
		Type:  "array",
		Items: data_type,
	}

	if required {
		arr.MinItems = IntPtr(1)
	}

	return arr
}

func (v *JSONSchemaVisitor) BeginRelation(name string) {
	v.current_element = name
}

func (v *JSONSchemaVisitor) VisitRelation(name string, _body any) any { //What type should the body be?
	if _body == nil { //If the body coalesced to null, this relation is readonly and can be ignored
		return nil
	}
	body := _body.(*jsonschema.Schema)

	return &namedSchema{name: name, schema: body}
}

func (v *JSONSchemaVisitor) BeginType(namespace string, name string) {
	v.required_fields = []string{}
}

// Construct type expression
func (v *JSONSchemaVisitor) VisitType(namespace string, name string, _relations []any, _data_fields []any) any {
	relations := to_typed_slice[namedSchema](_relations)
	data_fields := to_typed_slice[namedSchema](_data_fields)

	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
		Required:   v.required_fields,
	}

	for _, r := range relations {
		if r == nil {
			continue //Skip relations that coalesced to nil- they're readonly
		}

		schema.Properties[r.name] = r.schema
	}

	for _, f := range data_fields {
		schema.Properties[f.name] = f.schema
	}

	v.Schemas[name] = schema

	return &namedSchema{name: name, schema: schema}
}

func (v *JSONSchemaVisitor) BeginDataField(name string) {
	v.current_element = name
}

func (v *JSONSchemaVisitor) VisitDataField(name string, required bool, _data_type any) any {
	data_type := _data_type.(*jsonschema.Schema)
	if required {
		v.required_fields = append(v.required_fields, name)
	}

	return &namedSchema{
		name:   name,
		schema: data_type,
	}
}

func (v *JSONSchemaVisitor) VisitCompositeDataType(data_types []any) any {
	return &jsonschema.Schema{
		OneOf: to_typed_slice[jsonschema.Schema](data_types),
	}
}

func (v *JSONSchemaVisitor) VisitUUIDDataType() any {
	return &jsonschema.Schema{
		Type:   "string",
		Format: "uuid",
	}
}

func (v *JSONSchemaVisitor) VisitNumericIDDataType(min *int, max *int) any {
	schema := &jsonschema.Schema{Type: "integer"}

	schema.Minimum = intPtrToFloatPtr(min)
	schema.Maximum = intPtrToFloatPtr(max)

	return schema
}

func (v *JSONSchemaVisitor) VisitTextDataType(minLength *int, maxLength *int, regex *string) any {
	schema := &jsonschema.Schema{Type: "string"}

	schema.MinLength = minLength
	schema.MaxLength = maxLength
	if regex != nil {
		schema.Pattern = *regex
	}

	return schema
}

type namedSchema struct {
	name   string
	schema *jsonschema.Schema
}

func intPtrToFloatPtr(v *int) *float64 {
	if v == nil {
		return nil
	}

	f := float64(*v)

	return &f
}

func IntPtr(v int) *int {
	return &v
}

func to_typed_slice[T any](from []any) []*T {
	typed := make([]*T, len(from))
	for i, v := range from {
		if v == nil {
			typed[i] = nil
		} else {
			typed[i] = v.(*T)
		}
	}

	return typed
}
