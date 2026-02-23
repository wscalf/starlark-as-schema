package scripting

type SchemaVisitor interface {
	VisitAnd(left any, right any) any
	VisitOr(left any, right any) any
	VisitUnless(left any, right any) any
	VisitRelationExpression(name string) any
	VisitSubRelationExpression(name string, sub string) any
	VisitAssignableExpression(typeNamespace string, typeName string, cardinality string, data_type any) any

	BeginRelation(name string)
	// Construct relation expression
	VisitRelation(name string, body any) any

	BeginType(namespace string, name string)
	// Construct type expression
	VisitType(namespace string, name string, relations []any, data_fields []any) any

	BeginDataField(name string)
	VisitDataField(name string, required bool, data_type any) any

	VisitCompositeDataType(data_types []any) any
	VisitUUIDDataType() any
	VisitNumericIDDataType(min *int, max *int) any
	VisitTextDataType(minLength *int, maxLength *int, regex *string) any
}
