package main

import (
	"fmt"

	"github.com/stackrox/stackrox/central/graphql/resolvers"
)

func main() {
	fmt.Println(resolvers.Schema())
}
