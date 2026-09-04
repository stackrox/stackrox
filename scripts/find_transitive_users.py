#!/usr/bin/python3
import subprocess

# This script is useful for finding which packages in the StackRox Go codebase use (potentially transitively)
# a given set of dependency packages.

# Once we reach packages with these prefixes, we can terminate the dependency graph walk and report a finding.
OUR_PREFIXES = ['github.com/stackrox/rox', 'github.com/stackrox/scanner']


def main(pkg_spec, deps):
    importer_graph = load_importer_graph(pkg_spec)
    for dep in deps:
        found = []
        gather_import_paths(found, importer_graph, [dep])
        by_prefix = {}
        for path in found:
            path = list(reversed(path))
            if len(path) <= 2:
                print(" -> ".join(path))
            prefix = " -> ".join(path[:1])
            if prefix not in by_prefix or len(by_prefix[prefix]) > len(path):
                by_prefix[prefix] = path
        for path in by_prefix.values():
            print(" -> ".join(path))


def gather_import_paths(found, importer_graph, path, indent=0):
    dep = path[-1]
    if dep not in importer_graph:
        print('%s is not in the import graph' % dep)
        return
    if ourcode(dep):
        found.append(path)
        return
    for user in importer_graph[dep]:
        gather_import_paths(found, importer_graph, path[:]+[user], indent+1)


def load_importer_graph(pkg_spec):
    g = {}
    p = subprocess.run(["go", "list", "-deps", "-f", '{{ range $i := .Imports }}{{ $.ImportPath }} {{ $i }}{{ "\\n" }}{{ end }}', pkg_spec],
      capture_output=True, check=True, text=True)
    lines = p.stdout.split(sep="\n")
    for l in lines:
        l = l.strip()
        if not l:
            continue
        importer, imported = l.split(" ")
        if imported not in g:
            g[imported] = [importer]
        else:
            g[imported].append(importer)
    return g


def ourcode(pkg):
    for our_pkg in OUR_PREFIXES:
        if pkg.startswith(our_pkg):
            return True
    return False


if __name__ == '__main__':
    import sys
    if len(sys.argv) < 3:
        print("Usage: %s <pkg_spec> <dep_pkg> [ <dep_pkg> ... ]" % sys.argv[0])
        print("example: %s <pkg_spec> <dep_pkg> [ <dep_pkg> ... ]" % sys.argv[0])
        sys.exit(1)
    main(sys.argv[1], sys.argv[2:])
