package main

import (
	"fmt"

	"github.com/stackrox/rox/central/graphql/resolvers"
	_ "github.com/stackrox/rox/central/graphql/resolvers"
)

func main() {
	fmt.Println(resolvers.Schema())
}
