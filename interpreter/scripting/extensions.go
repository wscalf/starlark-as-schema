package scripting

import "go.starlark.net/starlark"

type Extensions struct {
	namespaces map[string]*Namespace
}

type Namespace struct {
	Overlay  map[string]map[string]starlark.Value
	Metadata map[string]starlark.Value
}

func NewExtensions() *Extensions {
	return &Extensions{
		namespaces: map[string]*Namespace{},
	}
}

func (e *Extensions) AddMember(namespace, typeName, relation string, body starlark.Value) {
	relations := e.ensureType(namespace, typeName)
	if _, ok := relations[relation]; !ok {
		relations[relation] = body
	}
}

func (e *Extensions) AddMetadata(namespace, key string, body starlark.Value) {
	ns := e.ensureNamespace(namespace)

	ns.Metadata[key] = body
}

func (e *Extensions) ensureNamespace(namespace string) *Namespace {
	var ok bool
	var ns *Namespace

	if ns, ok = e.namespaces[namespace]; !ok {
		ns = &Namespace{
			Overlay:  map[string]map[string]starlark.Value{},
			Metadata: map[string]starlark.Value{},
		}

		e.namespaces[namespace] = ns
	}

	return ns
}

func (e *Extensions) ensureType(namespace, typeName string) map[string]starlark.Value {
	ns := e.ensureNamespace(namespace)
	overlay := ns.Overlay

	var ok bool
	var relations map[string]starlark.Value
	if relations, ok = overlay[typeName]; !ok {
		relations = map[string]starlark.Value{}
		overlay[typeName] = relations
	}

	return relations
}
