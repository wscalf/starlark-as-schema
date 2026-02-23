package main

import (
	"encoding/json"
	"fmt"

	"example.com/interpreter/scripting"
	"example.com/interpreter/visitors"
	"go.starlark.net/starlark"
)

func main() {
	loader := scripting.NewLoader("schema")

	thread := &starlark.Thread{Name: "my thread", Load: loader.Load}

	moduleNames, err := loader.GetAllModuleNames()

	if err != nil {
		fmt.Println("Error listing contents of module directory: ", err)
		return
	}

	for _, name := range moduleNames {
		globals, err := loader.Load(thread, name)

		if err != nil {
			fmt.Println("Error processing module", name, ":", err)
			continue
		}

		printAll(globals)
	}

	namespaces, err := loader.BuildIntermediate(thread)
	if err != nil {
		fmt.Println("Error building intermediate model: ", err)
		return
	}

	visitModules := func(visitor scripting.SchemaVisitor) {
		for _, namespace := range namespaces {
			namespace.Visit(visitor)
		}
	}

	spiceDbVisitor := visitors.NewSpiceDBSchemaGeneratingVisitor()
	visitModules(spiceDbVisitor)

	schema, err := spiceDbVisitor.Generate()
	if err != nil {
		fmt.Println("Error generating SpiceDB schema:", err)
		return
	}
	fmt.Println("SpiceDB Schema:")
	fmt.Println(schema)

	fmt.Println("JSON Schemas per Type:")
	jsonSchemaVisitor := visitors.NewJSONSchemaVisitor()
	visitModules(jsonSchemaVisitor)
	for name, schema := range jsonSchemaVisitor.Schemas {
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			fmt.Println("error marshaling schema for type", name)
			continue
		}

		fmt.Println(name, ":")
		fmt.Println(string(data))
	}
}

func printAll(globals starlark.StringDict) {
	for k, v := range globals {
		fmt.Println(k, ":", v)
	}
}
