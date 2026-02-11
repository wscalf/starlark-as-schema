package scripting

import "go.starlark.net/starlark"

type Extensions struct {
	namespaces map[string]map[string]map[string]starlark.Value
}

func NewExtensions() *Extensions {
	return &Extensions{
		namespaces: map[string]map[string]map[string]starlark.Value{},
	}
}

func (e *Extensions) AddMember(namespace, typeName, relation string, body starlark.Value) {
	e.ensurePath(namespace, typeName)

	relations := e.namespaces[namespace][typeName]
	if _, ok := relations[relation]; !ok {
		relations[relation] = body
	}
}

func (e *Extensions) ensurePath(namespace, typeName string) {
	var types map[string]map[string]starlark.Value
	var ok bool
	if types, ok = e.namespaces[namespace]; !ok {
		types = map[string]map[string]starlark.Value{}
		e.namespaces[namespace] = types
	}

	var relations map[string]starlark.Value
	if relations, ok = types[typeName]; !ok {
		relations = map[string]starlark.Value{}
		types[typeName] = relations
	}
}
