package main

import (
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

	visitModules := func(visitor visitors.SchemaVisitor) error {
		for _, name := range moduleNames {
			err := loader.VisitModule(thread, name, visitor)
			if err != nil {
				return fmt.Errorf("error visiting module %s: %w", name, err)
			}
		}

		return nil
	}

	spiceDbVisitor := visitors.NewSpiceDBSchemaGeneratingVisitor()
	err = visitModules(spiceDbVisitor)
	if err != nil {
		fmt.Println(err)
		return
	}

	schema, err := spiceDbVisitor.Generate()
	if err != nil {
		fmt.Println("Error generating SpiceDB schema:", err)
		return
	}
	fmt.Println("SpiceDB Schema:")
	fmt.Println(schema)
}

func printAll(globals starlark.StringDict) {
	for k, v := range globals {
		fmt.Println(k, ":", v)
	}
}
