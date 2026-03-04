#!/usr/bin/env bash
set -euo pipefail

# This script checks that all go.mod files in the repository have a corresponding
# dependabot configuration, and that there are no orphaned dependabot configurations
# for go.mod files that no longer exist.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
cd "$ROOT"

# Find all go.mod files in the repository
mapfile -t gomod_files < <(find . -name "go.mod" -type f | sed 's|^\./||' | sort)

# Extract all gomod directories from dependabot.yaml
mapfile -t dependabot_dirs < <(yq e '.updates[] | select(.package-ecosystem=="gomod") | .directory' .github/dependabot.yaml | sort)

# Convert go.mod file paths to directory paths for comparison
declare -a gomod_dirs
for gomod_file in "${gomod_files[@]}"; do
    dir=$(dirname "$gomod_file")
    if [[ "$dir" == "." ]]; then
        gomod_dirs+=("/")
    else
        gomod_dirs+=("/$dir")
    fi
done

# Sort the gomod directories
IFS=$'\n' gomod_dirs=($(sort <<<"${gomod_dirs[*]}"))
unset IFS

# Check for go.mod files without dependabot configuration
exit_code=0
missing_configs=()
for dir in "${gomod_dirs[@]}"; do
    found=false
    for dep_dir in "${dependabot_dirs[@]}"; do
        # Normalize paths for comparison (remove trailing slash)
        normalized_dir="${dir%/}"
        normalized_dep="${dep_dir%/}"
        [[ "$normalized_dir" == "$normalized_dep" ]] && { found=true; break; }
    done
    if [[ "$found" == false ]]; then
        missing_configs+=("$dir")
    fi
done

if [[ ${#missing_configs[@]} -gt 0 ]]; then
    echo "ERROR: The following go.mod files do not have dependabot configurations:" >&2
    for dir in "${missing_configs[@]}"; do
        if [[ "$dir" == "/" ]]; then
            echo "  - ./go.mod (directory: /)" >&2
        else
            echo "  - .${dir}/go.mod (directory: ${dir})" >&2
        fi
    done
    echo "" >&2
    echo "Please add a gomod update entry in .github/dependabot.yaml for each missing directory." >&2
    exit_code=1
fi

# Check for orphaned dependabot configurations
orphaned_configs=()
for dep_dir in "${dependabot_dirs[@]}"; do
    found=false
    for dir in "${gomod_dirs[@]}"; do
        # Normalize paths for comparison (remove trailing slash)
        normalized_dir="${dir%/}"
        normalized_dep="${dep_dir%/}"
        [[ "$normalized_dir" == "$normalized_dep" ]] && { found=true; break; }
    done
    if [[ "$found" == false ]]; then
        orphaned_configs+=("$dep_dir")
    fi
done

if [[ ${#orphaned_configs[@]} -gt 0 ]]; then
    echo "ERROR: The following dependabot configurations refer to non-existent go.mod files:" >&2
    for dir in "${orphaned_configs[@]}"; do
        if [[ "$dir" == "/" ]]; then
            echo "  - directory: ${dir} (expected: ./go.mod)" >&2
        else
            echo "  - directory: ${dir} (expected: .${dir}/go.mod)" >&2
        fi
    done
    echo "" >&2
    echo "Please remove these stale entries from .github/dependabot.yaml." >&2
    exit_code=1
fi

if [[ $exit_code -eq 0 ]]; then
    echo "✓ All go.mod files have corresponding dependabot configurations."
    echo "✓ No orphaned dependabot configurations found."
fi

exit $exit_code
