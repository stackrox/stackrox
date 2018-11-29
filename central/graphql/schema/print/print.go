package main

import (
	"fmt"

	_ "github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/schema"
)

func main() {
	fmt.Println(schema.Schema())
}
