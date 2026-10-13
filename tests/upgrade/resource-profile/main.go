// resource-profile is a CI-only Helm post-renderer for the upgrade suites.
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	if err := render(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func render(in io.Reader, out io.Writer) error {
	decoder := yaml.NewDecoder(in)
	encoder := yaml.NewEncoder(out)
	defer encoder.Close()
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(doc.Content) == 0 {
			continue
		}
		root := doc.Content[0]
		kind := field(root, "kind")
		if kind != nil {
			switch kind.Value {
			case "Deployment", "DaemonSet", "StatefulSet":
				spec := field(field(field(root, "spec"), "template"), "spec")
				for _, name := range []string{"containers", "initContainers"} {
					containers := field(spec, name)
					if containers == nil {
						continue
					}
					for _, container := range containers.Content {
						var value map[string]interface{}
						if err := container.Decode(&value); err != nil {
							return err
						}
						if err := normalize(value); err != nil {
							return fmt.Errorf("%s %s: %w", kind.Value, name, err)
						}
						resources := field(container, "resources")
						if err := resources.Encode(value["resources"]); err != nil {
							return err
						}
					}
				}
			}
		}
		// Keep scalar tags and quoting outside resources intact. Decoding the
		// entire document into maps can change annotation value interpretation.
		if err := encoder.Encode(&doc); err != nil {
			return err
		}
	}
}

func field(node *yaml.Node, name string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == name {
			return node.Content[i+1]
		}
	}
	return nil
}

func object(value interface{}) map[string]interface{} {
	result, _ := value.(map[string]interface{})
	return result
}

func normalize(container map[string]interface{}) error {
	resources := object(container["resources"])
	requests := object(resources["requests"])
	if requests["cpu"] == nil {
		return fmt.Errorf("container %v is missing CPU request", container["name"])
	}
	cpu := fmt.Sprint(requests["cpu"])
	multiplier := 1000.0
	if strings.HasSuffix(cpu, "m") {
		cpu = strings.TrimSuffix(cpu, "m")
		multiplier = 1
	}
	quantity, err := strconv.ParseFloat(cpu, 64)
	if err != nil {
		return fmt.Errorf("invalid CPU request: %w", err)
	}
	if quantity*multiplier > 500 {
		requests["cpu"] = "500m"
	}
	delete(object(resources["limits"]), "cpu")
	return nil
}
