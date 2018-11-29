package main

import (
	"bytes"
	"io/ioutil"

	"github.com/stackrox/rox/central/graphql/generator"
	"github.com/stackrox/rox/central/graphql/schema"
	_ "github.com/stackrox/rox/generated/api/v1"
)

func main() {
	w := &bytes.Buffer{}
	generator.GenerateResolvers(schema.WalkParameters, w)
	err := ioutil.WriteFile("generated.go", w.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}
