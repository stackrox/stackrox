#!/usr/bin/env python3
"""Wire cloud GOCACHE backend into all cache-go-dependencies call sites.

Line-based and indentation-aware: for each `uses: ./.github/actions/cache-go-dependencies`
line, add gocache-backend/azure inputs to its with: block (creating the
block if absent), respecting the call site's indentation.
"""
import sys

USES = "uses: ./.github/actions/cache-go-dependencies"
CLOUD_KEYS = [
    "gocache-backend: cloud",
    "azure-tenant-id: ${{ secrets.AZURE_TENANT_ID }}",
    "azure-client-id: ${{ secrets.AZURE_CLIENT_ID }}",
    "azure-client-secret: ${{ secrets.AZURE_CLIENT_SECRET }}",
]

def wire(path):
    lines = open(path).readlines()
    out = []
    i = 0
    wired = 0
    while i < len(lines):
        line = lines[i]
        out.append(line)
        if line.strip() == USES:
            indent = line[: len(line) - len(line.lstrip())]
            j = i + 1
            has_with = (
                j < len(lines)
                and lines[j].strip() == "with:"
                and lines[j].startswith(indent + "with:")
            )
            if has_with:
                out.append(lines[j])
                j += 1
                while j < len(lines) and lines[j].startswith(indent + "  ") and lines[j].strip():
                    out.append(lines[j])
                    j += 1
                i = j - 1
            else:
                out.append(indent + "with:\n")
            for key in CLOUD_KEYS:
                out.append(indent + "  " + key + "\n")
            wired += 1
        i += 1
    if wired:
        open(path, "w").write("".join(out))
    print(f"{path}: {wired} call sites wired")

if __name__ == "__main__":
    for f in sys.argv[1:]:
        wire(f)
