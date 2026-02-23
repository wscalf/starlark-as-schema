package scripting

// After lunch:
// - Add constructors and call them from Starlark
// - How do type unions work?
// - Replace explicit visitor traversal with gather (build intermediate model from Starlark explicit + extensions), then visit

type Namespace struct {
	Name  string
	Types map[string]*ResourceType
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:  name,
		Types: map[string]*ResourceType{},
	}
}

func (n *Namespace) Visit(visitor SchemaVisitor) {
	for typeName, resource_type := range n.Types {
		resource_type.Visit(n.Name, typeName, visitor)
	}
}

type ResourceType struct {
	Relations map[string]*Relation
	Fields    map[string]*Field
}

func NewResourceType(namespace, name string) *ResourceType {
	return &ResourceType{
		Relations: map[string]*Relation{},
		Fields:    map[string]*Field{},
	}
}

func (t *ResourceType) Visit(namespace, typeName string, visitor SchemaVisitor) any {
	visitor.BeginType(namespace, typeName)

	convertedRelations := make([]any, 0, len(t.Relations))
	for name, obj := range t.Relations {
		convertedRelations = append(convertedRelations, obj.Visit(name, visitor))
	}

	convertedFields := make([]any, 0, len(t.Fields))
	for name, obj := range t.Fields {
		convertedFields = append(convertedFields, obj.Visit(name, visitor))
	}

	return visitor.VisitType(namespace, typeName, convertedRelations, convertedFields)
}

type Relation struct {
	Body Visitable
}

func NewRelation(body Visitable) *Relation {
	return &Relation{
		Body: body,
	}
}

func (r *Relation) Visit(name string, visitor SchemaVisitor) any {
	visitor.BeginRelation(name)

	body := r.Body.Visit(visitor)
	return visitor.VisitRelation(name, body)
}

type SetOperation struct {
	Kind  string
	Left  Visitable
	Right Visitable
}

func NewSetOperation(kind string, left Visitable, right Visitable) *SetOperation {
	return &SetOperation{
		Kind:  kind,
		Left:  left,
		Right: right,
	}
}

func (s *SetOperation) Visit(visitor SchemaVisitor) any {
	convertedLeft := s.Left.Visit(visitor)
	convertedRight := s.Right.Visit(visitor)

	switch s.Kind {
	case "and":
		return visitor.VisitAnd(convertedLeft, convertedRight)
	case "or":
		return visitor.VisitOr(convertedLeft, convertedRight)
	case "unless":
		return visitor.VisitUnless(convertedLeft, convertedRight)
	default:
		panic("unmatched set operation kind: " + s.Kind)
	}
}

type RelationRef struct {
	Name    string
	SubName *string
}

func NewRelationRef(name string, subname *string) *RelationRef {
	return &RelationRef{
		Name:    name,
		SubName: subname,
	}
}

func (r *RelationRef) Visit(visitor SchemaVisitor) any {
	if r.SubName != nil {
		return visitor.VisitSubRelationExpression(r.Name, *r.SubName)
	} else {
		return visitor.VisitRelationExpression(r.Name)
	}
}

type Assignable struct {
	Namespace   string
	Type        string
	Cardinality string
	DataType    Visitable
}

func NewAssignable(namespace, resource_type, cardinality string, data_type Visitable) *Assignable {
	return &Assignable{
		Namespace:   namespace,
		Type:        resource_type,
		Cardinality: cardinality,
		DataType:    data_type,
	}
}

func (a *Assignable) Visit(visitor SchemaVisitor) any {
	dataType := a.DataType.Visit(visitor)
	return visitor.VisitAssignableExpression(a.Namespace, a.Type, a.Cardinality, dataType)
}

type Field struct {
	DataType Visitable
	Required bool
}

func NewField(data_type Visitable, required bool) *Field {
	return &Field{
		DataType: data_type,
		Required: required,
	}
}

func (f *Field) Visit(name string, visitor SchemaVisitor) any {
	visitor.BeginDataField(name)

	dataType := f.DataType.Visit(visitor)

	return visitor.VisitDataField(name, f.Required, dataType)
}

type TextType struct {
	MinLength *int
	MaxLength *int
	Regex     *string
}

func NewTextType(min_length, max_length *int, regex *string) *TextType {
	return &TextType{
		MinLength: min_length,
		MaxLength: max_length,
		Regex:     regex,
	}
}

func (t *TextType) Visit(visitor SchemaVisitor) any {
	return visitor.VisitTextDataType(t.MinLength, t.MaxLength, t.Regex)
}

type NumericIDType struct {
	Min *int
	Max *int
}

func NewNumericIDType(min, max *int) *NumericIDType {
	return &NumericIDType{
		Min: min,
		Max: max,
	}
}

func (n *NumericIDType) Visit(visitor SchemaVisitor) any {
	return visitor.VisitNumericIDDataType(n.Min, n.Max)
}

type UUIDType struct{}

func NewUUIDType() *UUIDType {
	return &UUIDType{}
}

func (u *UUIDType) Visit(visitor SchemaVisitor) any {
	return visitor.VisitUUIDDataType()
}

type TypeUnion struct {
	Left  Visitable
	Right Visitable
}

func NewTypeUnion(left, right Visitable) *TypeUnion {
	return &TypeUnion{
		Left:  left,
		Right: right,
	}
}

func (t *TypeUnion) Visit(visitor SchemaVisitor) any {
	left := t.Left.Visit(visitor)
	right := t.Right.Visit(visitor)

	return visitor.VisitCompositeDataType([]any{left, right})
}

type Visitable interface {
	Visit(visitor SchemaVisitor) any
}
