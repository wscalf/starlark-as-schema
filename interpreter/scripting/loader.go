package scripting

import (
	"fmt"
	"os"
	"path"
	"strings"

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

func (l *Loader) BuildIntermediate(thread *starlark.Thread) ([]*Namespace, error) {
	moduleNames, err := l.GetAllModuleNames()
	if err != nil {
		return nil, err
	}

	namespaces := []*Namespace{}

	for _, moduleName := range moduleNames {
		globals, err := l.Load(thread, moduleName)
		if err != nil {
			return nil, err
		}
		namespaceName := strings.TrimSuffix(moduleName, ".star")
		namespace := NewNamespace(namespaceName)

		moduleTypeExtents, moduleTypeExtentsFound := l.extensions.namespaces[namespaceName]

		for typeName, members := range globals {
			typeObj := NewResourceType(namespaceName, typeName)

			relations := map[string]*Relation{}
			fields := map[string]*Field{}

			convertMember := func(memberName string, memberData *starlarkstruct.Struct) error {
				kind, err := get_string("kind", memberData)
				if err != nil {
					return err
				}
				switch kind {
				case "relation":
					r, err := convertRelation(memberData)
					if err != nil {
						return err
					}
					relations[memberName] = r
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

					converted_type, err := convertDataType(typeData)
					if err != nil {
						return err
					}

					fields[memberName] = NewField(converted_type, required)
					return nil
				default:
					return fmt.Errorf("unmatched 'kind' value in member %s of type extension %s from namespace %s: %s", memberName, typeName, moduleName, string(kind))
				}
			}

			if moduleTypeExtentsFound {
				if typeExtent, ok := moduleTypeExtents[typeName]; ok {
					for memberName, member := range typeExtent {
						memberData := member.(*starlarkstruct.Struct)

						err = convertMember(memberName, memberData)
						if err != nil {
							return nil, err
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
				err = convertMember(memberName, memberData)
				if err != nil {
					return nil, err
				}
			}

			typeObj.Fields = fields
			typeObj.Relations = relations

			namespace.Types[typeName] = typeObj
		}

		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

func convertRelation(memberData *starlarkstruct.Struct) (*Relation, error) {
	bodyData, err := memberData.Attr("body")
	if err != nil {
		return nil, err
	}
	body, err := convertRelationBody(bodyData)
	if err != nil {
		return nil, err
	}

	return NewRelation(body), nil
}

func convertDataType(v starlark.Value) (Visitable, error) {
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
		left, err := convertDataType(leftData)

		rightData, err := typeData.Attr("right")
		if err != nil {
			return nil, err
		}
		right, err := convertDataType(rightData)

		return NewTypeUnion(left, right), nil
	case "uuid":
		return NewUUIDType(), nil
	case "numeric_id":
		min, err := get_optional_number("min", typeData)
		if err != nil {
			return nil, err
		}

		max, err := get_optional_number("max", typeData)
		if err != nil {
			return nil, err
		}

		return NewNumericIDType(min, max), nil
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

		return NewTextType(minLength, maxLength, regex), nil
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

func convertRelationBody(v starlark.Value) (Visitable, error) {
	bodyData := v.(*starlarkstruct.Struct)
	kind, err := get_string("kind", bodyData)
	if err != nil {
		return nil, err
	}

	switch kind {
	case "and":
		left, right, err := convertBinaryArgumentsInRelationBody(bodyData)
		if err != nil {
			return nil, err
		}

		return NewSetOperation("and", left, right), nil
	case "or":
		left, right, err := convertBinaryArgumentsInRelationBody(bodyData)
		if err != nil {
			return nil, err
		}

		return NewSetOperation("or", left, right), nil
	case "unless":
		left, right, err := convertBinaryArgumentsInRelationBody(bodyData)
		if err != nil {
			return nil, err
		}

		return NewSetOperation("unless", left, right), nil
	case "ref":
		n, err := get_string("name", bodyData)
		if err != nil {
			return nil, err
		}

		return NewRelationRef(n, nil), nil
	case "subref":
		n, err := get_string("name", bodyData)
		if err != nil {
			return nil, err
		}
		sub, err := get_string("subname", bodyData)
		if err != nil {
			return nil, err
		}

		return NewRelationRef(n, &sub), nil
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

		data_type, err := convertDataType(typeData)
		if err != nil {
			return nil, err
		}

		return NewAssignable(ns, tn, cardinality, data_type), nil
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

func convertBinaryArgumentsInRelationBody(body *starlarkstruct.Struct) (left Visitable, right Visitable, err error) {
	leftData, err := body.Attr("left")
	if err != nil {
		return
	}
	left, err = convertRelationBody(leftData)

	rightData, err := body.Attr("right")
	if err != nil {
		return
	}
	right, err = convertRelationBody(rightData)
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
			return nil, err
		}

		t, err := convert_to_string(args.Index(1))
		if err != nil {
			return nil, err
		}

		r, err := convert_to_string(args.Index(2))
		if err != nil {
			return nil, err
		}

		b := args.Index(3)

		l.extensions.AddMember(ns, t, r, b)

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
