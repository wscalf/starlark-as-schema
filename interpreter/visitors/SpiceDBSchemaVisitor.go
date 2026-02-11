package visitors

import (
	"github.com/authzed/spicedb/pkg/namespace"
	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
	"github.com/authzed/spicedb/pkg/schemadsl/compiler"
	"github.com/authzed/spicedb/pkg/schemadsl/generator"
)

type SpiceDBSchemaGeneratingVisitor struct {
	elements             []compiler.SchemaDefinition
	tuple_relations      []*corev1.Relation
	currentRelationName  string
	currentTypeNamespace string
	currentTypeName      string
}

func NewSpiceDBSchemaGeneratingVisitor() *SpiceDBSchemaGeneratingVisitor {
	return &SpiceDBSchemaGeneratingVisitor{
		elements:        []compiler.SchemaDefinition{},
		tuple_relations: []*corev1.Relation{},
	}
}

func (v *SpiceDBSchemaGeneratingVisitor) Generate() (string, error) {
	data, _, err := generator.GenerateSchema(v.elements)
	return data, err
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitAnd(left any, right any) any {
	rewrite := namespace.Intersection(left.(*corev1.SetOperation_Child), right.(*corev1.SetOperation_Child))
	return namespace.Rewrite(rewrite)
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitOr(left any, right any) any {
	rewrite := namespace.Union(left.(*corev1.SetOperation_Child), right.(*corev1.SetOperation_Child))
	return namespace.Rewrite(rewrite)
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitUnless(left any, right any) any {
	rewrite := namespace.Exclusion(left.(*corev1.SetOperation_Child), right.(*corev1.SetOperation_Child))
	return namespace.Rewrite(rewrite)
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitRelationExpression(name string) any {
	return namespace.ComputedUserset(name)
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitSubRelationExpression(name string, sub string) any {
	return namespace.TupleToUserset(tuple_relation_name_from_relation_name(name), sub) //For rel->subrel expressions, use the tuple relation name
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitAssignableExpression(typeNamespace string, typeName string, cardinality string, data_type any) any {
	tuple_relation_name := tuple_relation_name_from_relation_name(v.currentRelationName)

	var allowed_relation *corev1.AllowedRelation
	if cardinality == "All" {
		allowed_relation = namespace.AllowedPublicNamespace(spiceDBTypeName(typeNamespace, typeName))
	} else {
		allowed_relation = namespace.AllowedRelation(spiceDBTypeName(typeNamespace, typeName), compiler.Ellipsis) //Note- need to handle subrelations like group.member here
	}

	tuple_relation := namespace.MustRelation(tuple_relation_name, nil, allowed_relation)
	v.tuple_relations = append(v.tuple_relations, tuple_relation)
	return namespace.ComputedUserset(tuple_relation_name)
}

func (v *SpiceDBSchemaGeneratingVisitor) BeginRelation(name string) {
	v.currentRelationName = name
}

// Construct relation expression
func (v *SpiceDBSchemaGeneratingVisitor) VisitRelation(name string, body any) any {
	return namespace.MustRelation(name, namespace.Union(body.(*corev1.SetOperation_Child)))
}

func (v *SpiceDBSchemaGeneratingVisitor) BeginType(namespace string, name string) {
	v.currentTypeNamespace = namespace
	v.currentTypeName = name
	v.tuple_relations = []*corev1.Relation{}
}

// Construct type expression
func (v *SpiceDBSchemaGeneratingVisitor) VisitType(ns string, name string, relations []any, _ []any) any {
	zanzibar_namespace := namespace.Namespace(spiceDBTypeName(ns, name)) //Need to handle namespace here

	converted_relations := make([]*corev1.Relation, len(relations))
	for i, r := range relations {
		converted_relations[i] = r.(*corev1.Relation)
	}

	zanzibar_namespace.Relation = append(converted_relations, v.tuple_relations...)

	v.elements = append(v.elements, zanzibar_namespace)

	return zanzibar_namespace
}

// SpiceDB schema doesn't reflect data types
func (v *SpiceDBSchemaGeneratingVisitor) BeginDataField(name string) {

}

func (v *SpiceDBSchemaGeneratingVisitor) VisitDataField(name string, required bool, data_type any) any {
	return nil
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitCompositeDataType(data_types []any) any {
	return nil
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitUUIDDataType() any {
	return nil
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitNumericIDDataType(min *int, max *int) any {
	return nil
}

func (v *SpiceDBSchemaGeneratingVisitor) VisitTextDataType(minLength *int, maxLength *int, regex *string) any {
	return nil
}

func tuple_relation_name_from_relation_name(name string) string {
	return "t_" + name
}

func spiceDBTypeName(ns string, name string) string {
	return ns + "/" + name
}
