package common

import (
	"github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurefmt"
)

type libDepNode struct {
	// Features used by this library.
	features FeatureKeySet
	// Libraries used by this library directly.
	libraries set.StringSet
	// True if this node has been visited and features are fully populated.
	completed bool
}

// cycle is discovered while traversing a graph of dependency.
// head: the first element that was traversed within the cycle.
// members: all the elements that may appear in the cycle regardless of the order.
type cycle struct {
	head    string
	members set.StringSet
}

// GetDepMapRHEL creates a dependency map from a library to the features it uses.
func GetDepMapRHEL(pkgEnvs map[int]*database.RHELv2PackageEnv) map[string]FeatureKeySet {
	// Map from a library to its dependency data
	libNodes := make(map[string]*libDepNode)
	// Build the map
	for _, pkgEnv := range pkgEnvs {
		fvKey := featurefmt.PackageKey{
			Name:    pkgEnv.Pkg.Name,
			Version: pkgEnv.Pkg.GetPackageVersion(),
		}
		// Populate libraries with all direct imports.
		for lib, deps := range pkgEnv.Pkg.LibraryToDependencies {
			if node, ok := libNodes[lib]; ok {
				node.libraries = node.libraries.Union(deps)
				node.features.Add(fvKey)
			} else {
				node = &libDepNode{
					libraries: deps,
					features:  FeatureKeySet{fvKey: {}},
				}
				libNodes[lib] = node
			}
		}
	}
	return createDepMap(libNodes)
}

// GetDepMap creates a dependency map from a library to the features it uses.
func GetDepMap(features []database.FeatureVersion) map[string]FeatureKeySet {
	// Map from a library to its dependency data
	libNodes := make(map[string]*libDepNode)
	// Build the map
	for _, feature := range features {
		fvKey := featurefmt.PackageKey{
			Name:    feature.Feature.Name,
			Version: feature.Version,
		}
		// Populate libraries with all direct imports.
		for lib, deps := range feature.LibraryToDependencies {
			if node, ok := libNodes[lib]; ok {
				node.libraries = node.libraries.Union(deps)
				node.features.Add(fvKey)
			} else {
				node = &libDepNode{
					libraries: deps,
					features:  FeatureKeySet{fvKey: {}},
				}
				libNodes[lib] = node
			}
		}
	}
	return createDepMap(libNodes)
}

// Traverse map of lib dep nodes and create a dependency map
func createDepMap(libNodes map[string]*libDepNode) map[string]FeatureKeySet {
	depMap := make(map[string]FeatureKeySet)
	for k, v := range libNodes {
		var c *cycle
		depMap[k], c = fillIn(libNodes, k, v, map[string]int{k: 0})
		if c != nil {
			// This is a very rare case that we have a loop in dependency map.
			// All members in the loop should map to the same set of features.
			for c := range c.members {
				depMap[c] = depMap[k]
			}
		}
	}
	return depMap
}

func fillIn(libToDep map[string]*libDepNode, depname string, dep *libDepNode, path map[string]int) (FeatureKeySet, *cycle) {
	if dep.completed {
		return dep.features, nil
	}
	var cycles []cycle
	for lib := range dep.libraries {
		execs, ok := libToDep[lib]
		if !ok {
			logrus.Debugf("Unresolved soname %s", lib)
			continue
		}
		if seq, ok := path[lib]; ok {
			// This is a very rare case that we detect a loop in dependency map.
			// We create a cycle and put it in the cycles.
			// We use a map from library to its sequence number in path to prioritize the most frequently used code path.
			c := cycle{head: lib, members: set.NewStringSet(lib)}
			for p, s := range path {
				if s > seq {
					c.members.Add(p)
				}
			}
			cycles = append(cycles, c)
			continue
		}
		path[lib] = len(path)
		features, c := fillIn(libToDep, lib, execs, path)
		delete(path, lib)
		if c != nil {
			cycles = append(cycles, *c)
		}
		dep.features.Merge(features)
	}
	dep.completed = true
	if len(cycles) == 0 {
		return dep.features, nil
	}

	// Again, this is a rare case that we have a cycle in the dependency graph.
	mc := cycle{head: depname, members: set.NewStringSet()}
	for _, c := range cycles {
		// This is an extremely rare case we have multiple cycles.
		// Merge multiple cycles together to form a bigger possible cycle.
		if path[c.head] < path[mc.head] {
			mc.head = c.head
		}
		mc.members = mc.members.Union(c.members)
	}
	// If this is the head of the cycle, resolve the cycle by assigning the features
	// of the head to all members
	if mc.head == depname {
		for c := range mc.members {
			libToDep[c].features = dep.features
		}
		return dep.features, nil
	}
	return dep.features, &mc
}
