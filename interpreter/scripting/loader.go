package scripting

import (
	"fmt"
	"os"
	"path"
	"strings"

	"example.com/interpreter/visitors"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

type Loader struct {
	path        string
	modules     map[string]starlark.StringDict
	opts        *syntax.FileOptions
	predeclared starlark.StringDict
	extensions  *Extensions
}

func (l *Loader) Load(thread *starlark.Thread, name string) (starlark.StringDict, error) {
	if m, ok := l.modules[name]; ok {
		return m, nil
	}

	location := path.Join(l.path, name)
	contents, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	globals, err := starlark.ExecFileOptions(l.opts, thread, name, contents, l.predeclared)
	if err != nil {
		return nil, err
	}

	l.modules[name] = globals

	return globals, nil
}

func (l *Loader) VisitMetadata(visitor func(string, map[string]starlark.Value) error) error {
	for name, ns := range l.extensions.namespaces {
		if err := visitor(name, ns.Metadata); err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) VisitModule(thread *starlark.Thread, name string, visitor visitors.SchemaVisitor) error {
	globals, err := l.Load(thread, name)
	if err != nil {
		return err
	}
	moduleName := strings.TrimSuffix(name, ".star")

	moduleTypeExtents, moduleTypeExtentsFound := l.extensions.namespaces[moduleName]

	for typeName, members := range globals {
		visitor.BeginType(moduleName, typeName)
		relations := []any{}
		fields := []any{}

		visitMember := func(typeName string, memberName string, memberData *starlarkstruct.Struct) error {
			kind, err := get_string("kind", memberData)
			if err != nil {
				return err
			}
			switch kind {
			case "relation":
				r, err := visitRelation(visitor, memberName, memberData)
				if err != nil {
					return err
				}
				relations = append(relations, r)
				return nil
			case "field":
				typeData, err := memberData.Attr("type")
				if err != nil {
					return fmt.Errorf("error accessing 'type' member of %+v: %w", typeData, err)
				}

				required, err := get_bool("required", memberData)
				if err != nil {
					return err
				}

				converted_type, err := visitDataType(visitor, typeData)
				if err != nil {
					return err
				}

				fields = append(fields, visitor.VisitDataField(memberName, required, converted_type))
				return nil
			default:
				return fmt.Errorf("unmatched 'kind' value in member %s of type extension %s from namespace %s: %s", memberName, typeName, moduleName, string(kind))
			}
		}

		if moduleTypeExtentsFound {
			if typeExtent, ok := moduleTypeExtents.Overlay[typeName]; ok {
				for memberName, member := range typeExtent {
					memberData := member.(*starlarkstruct.Struct)

					err = visitMember(typeName, memberName, memberData)
					if err != nil {
						return err
					}
				}
			}
		}

		membersData, ok := members.(*starlark.Dict)
		if !ok {
			continue //Not a dict, not a type
		}

		for n, v := range membersData.Entries() {
			memberName := string(n.(starlark.String))
			memberData := v.(*starlarkstruct.Struct)
			err = visitMember(typeName, memberName, memberData)
			if err != nil {
				return err
			}
		}

		visitor.VisitType(moduleName, typeName, relations, fields)
	}

	return nil
}

func visitRelation(visitor visitors.SchemaVisitor, relationName string, memberData *starlarkstruct.Struct) (any, error) {
	visitor.BeginRelation(relationName)

	bodyData, err := memberData.Attr("body")
	if err != nil {
		return nil, err
	}
	body, err := visitRelationBody(visitor, bodyData)
	if err != nil {
		return nil, err
	}

	return visitor.VisitRelation(relationName, body), nil
}

func visitDataType(visitor visitors.SchemaVisitor, v starlark.Value) (any, error) {
	typeData, ok := v.(*starlarkstruct.Struct)
	if !ok {
		return nil, fmt.Errorf("error converting input of type %s to a starlark struct", v.Type())
	}

	kind, err := get_string("kind", typeData)
	if err != nil {
		return nil, err
	}

	switch kind {
	case "or": //Type union
		leftData, err := typeData.Attr("left")
		if err != nil {
			return nil, err
		}
		left, err := visitDataType(visitor, leftData)

		rightData, err := typeData.Attr("right")
		if err != nil {
			return nil, err
		}
		right, err := visitDataType(visitor, rightData)

		return visitor.VisitCompositeDataType([]any{left, right}), nil
	case "uuid":
		return visitor.VisitUUIDDataType(), nil
	case "numeric_id":
		min, err := get_optional_number("min", typeData)
		if err != nil {
			return nil, err
		}

		max, err := get_optional_number("max", typeData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitNumericIDDataType(min, max), nil
	case "text":
		minLength, err := get_optional_number("minLength", typeData)
		if err != nil {
			return nil, err
		}

		maxLength, err := get_optional_number("maxLength", typeData)
		if err != nil {
			return nil, err
		}

		regex, err := get_optional_string("regex", typeData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitTextDataType(minLength, maxLength, regex), nil
	default:
		return nil, fmt.Errorf("unmatched data type kind: %s", kind)
	}
}

func get_optional_number(name string, structure *starlarkstruct.Struct) (*int, error) {
	v, err := structure.Attr(name)
	if err != nil {
		return nil, fmt.Errorf("error accessing member %s of struct %+v: %w", name, structure, err)
	}

	if _, ok := v.(starlark.NoneType); ok {
		return nil, nil
	}

	if i, ok := v.(starlark.Int); ok {
		n, _ := i.Int64()
		r := int(n)
		return &r, nil
	} else {
		return nil, fmt.Errorf("unable to convert Starlark value of type %s to int", v.Type())
	}
}

func visitRelationBody(visitor visitors.SchemaVisitor, v starlark.Value) (any, error) {
	bodyData := v.(*starlarkstruct.Struct)
	kind, err := get_string("kind", bodyData)
	if err != nil {
		return err, nil
	}

	switch kind {
	case "and":
		left, right, err := visitBinaryArgumentsInRelationBody(visitor, bodyData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitAnd(left, right), nil
	case "or":
		left, right, err := visitBinaryArgumentsInRelationBody(visitor, bodyData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitOr(left, right), nil
	case "unless":
		left, right, err := visitBinaryArgumentsInRelationBody(visitor, bodyData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitUnless(left, right), nil
	case "ref":
		n, err := get_string("name", bodyData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitRelationExpression(n), nil
	case "subref":
		n, err := get_string("name", bodyData)
		if err != nil {
			return nil, err
		}
		sub, err := get_string("subname", bodyData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitSubRelationExpression(n, sub), nil
	case "assignable":
		ns, err := get_string("namespace", bodyData)
		if err != nil {
			return nil, err
		}
		tn, err := get_string("type", bodyData)
		if err != nil {
			return nil, err
		}
		cardinality, err := get_string("cardinality", bodyData)
		if err != nil {
			return nil, err
		}

		typeData, err := bodyData.Attr("data_type")
		if err != nil {
			return nil, err
		}

		data_type, err := visitDataType(visitor, typeData)
		if err != nil {
			return nil, err
		}

		return visitor.VisitAssignableExpression(ns, tn, cardinality, data_type), nil
	default:
		return nil, fmt.Errorf("unmatched relation expression kind: %s", kind)
	}
}

func get_string(name string, structure *starlarkstruct.Struct) (string, error) {
	v, err := structure.Attr(name)
	if err != nil {
		return "", fmt.Errorf("error accessing member %s of struct %+v: %w", name, structure, err)
	}

	return convert_to_string(v)
}

func get_optional_string(name string, structure *starlarkstruct.Struct) (*string, error) {
	v, err := structure.Attr(name)
	if err != nil {
		return nil, fmt.Errorf("error accessing member %s of struct %+v: %w", name, structure, err)
	}

	if _, ok := v.(starlark.NoneType); ok {
		return nil, nil
	}

	s, err := convert_to_string(v)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func convert_to_string(v starlark.Value) (string, error) {
	if s, ok := v.(starlark.String); ok {
		return string(s), nil
	} else {
		return "", fmt.Errorf("unable to convert Starlark value of type %s to string", v.Type())
	}
}

func convert_to_callable(v starlark.Value) (starlark.Callable, error) {
	if c, ok := v.(starlark.Callable); ok {
		return c, nil
	} else {
		return nil, fmt.Errorf("unable to convert Starlark value of type %s to Callable", v.Type())
	}
}

func get_bool(name string, structure *starlarkstruct.Struct) (bool, error) {
	v, err := structure.Attr(name)
	if err != nil {
		return false, fmt.Errorf("error access member %s of struct %+v: %w", name, structure, err)
	}

	if b, ok := v.(starlark.Bool); ok {
		return bool(b), nil
	} else {
		return false, fmt.Errorf("unable to convert Starlark value of type %s to bool", v.Type())
	}
}

func visitBinaryArgumentsInRelationBody(visitor visitors.SchemaVisitor, body *starlarkstruct.Struct) (left any, right any, err error) {
	leftData, err := body.Attr("left")
	if err != nil {
		return
	}
	left, err = visitRelationBody(visitor, leftData)

	rightData, err := body.Attr("right")
	if err != nil {
		return
	}
	right, err = visitRelationBody(visitor, rightData)
	return
}

func (l *Loader) GetAllModuleNames() ([]string, error) {
	entries, err := os.ReadDir(l.path)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}

func (l *Loader) RegisterBuiltin(name string, callback func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)) {
	v := starlark.NewBuiltin(name, callback)
	l.predeclared[name] = v
}

func (l *Loader) registerDefaultBuiltins() {
	l.RegisterBuiltin("struct", starlarkstruct.Make)

	l.RegisterBuiltin("add_member", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		ns, err := convert_to_string(args.Index(0))
		if err != nil {
			return starlark.None, err
		}

		t, err := convert_to_string(args.Index(1))
		if err != nil {
			return starlark.None, err
		}

		r, err := convert_to_string(args.Index(2))
		if err != nil {
			return starlark.None, err
		}

		b := args.Index(3)

		l.extensions.AddMember(ns, t, r, b)

		return starlark.None, nil
	})

	l.RegisterBuiltin("has_member", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		ns, err := convert_to_string(args.Index(0))
		if err != nil {
			return starlark.None, err
		}

		t, err := convert_to_string(args.Index(1))
		if err != nil {
			return starlark.None, err
		}

		r, err := convert_to_string(args.Index(2))
		if err != nil {
			return starlark.None, err
		}

		if namespace, ok := l.extensions.namespaces[ns]; ok {
			if _type, ok := namespace.Overlay[t]; ok {
				if _, ok := _type[r]; ok {
					return starlark.True, nil
				}
			}
		}

		return starlark.False, nil
	})

	l.RegisterBuiltin("replace_member", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		ns, err := convert_to_string(args.Index(0))
		if err != nil {
			return starlark.None, err
		}

		t, err := convert_to_string(args.Index(1))
		if err != nil {
			return starlark.None, err
		}

		r, err := convert_to_string(args.Index(2))
		if err != nil {
			return starlark.None, err
		}

		mutator, err := convert_to_callable(args.Index(3))
		if err != nil {
			return nil, err
		}

		var relation *starlarkstruct.Struct
		if namespace, ok := l.extensions.namespaces[ns]; ok {
			if _type, ok := namespace.Overlay[t]; ok {
				if r, ok := _type[r]; ok {
					relation = r.(*starlarkstruct.Struct)
				} else {
					return starlark.None, fmt.Errorf("relation %s does not exist on type %s in module %s", r, t, ns)
				}
			}
		}

		body, err := relation.Attr("body")
		if err != nil {
			return starlark.None, err
		}

		updated_body, err := starlark.Call(thread, mutator, starlark.Tuple{body}, []starlark.Tuple{})
		if err != nil {
			return starlark.None, err
		}

		updated, err := starlarkstruct.Make(nil, nil, starlark.Tuple{}, []starlark.Tuple{
			{starlark.String("kind"), starlark.String("relation")},
			{starlark.String("body"), updated_body},
		})

		if err != nil {
			return starlark.None, err
		}

		l.extensions.namespaces[ns].Overlay[t][r] = updated
		return starlark.None, nil
	})

	l.RegisterBuiltin("add_metadata", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		ns, err := convert_to_string(args.Index(0))
		if err != nil {
			return starlark.None, err
		}

		key, err := convert_to_string(args.Index(1))
		if err != nil {
			return starlark.None, err
		}

		value := args.Index(2)

		l.extensions.AddMetadata(ns, key, value)

		return starlark.None, nil
	})
}

func NewLoader(path string) *Loader {
	l := &Loader{
		path:        path,
		modules:     map[string]starlark.StringDict{},
		opts:        &syntax.FileOptions{},
		predeclared: starlark.StringDict{},
		extensions:  NewExtensions(),
	}

	l.registerDefaultBuiltins()

	return l
}
