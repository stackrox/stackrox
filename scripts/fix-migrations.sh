#!/usr/bin/env bash

PACKAGES=${MIGRATION_PACKAGES//,/ }

TEMP_DIR="/tmp/stackrox/migration-fixer"
if [[ -d "$TEMP_DIR" ]]; then
    rm -rf "$TEMP_DIR"
fi
mkdir -p "$TEMP_DIR"

for package in $PACKAGES; do
    MIGRATION_PATH="migrator/migrations/${package}"

    if ! [[ -d "$MIGRATION_PATH" ]]; then
        echo "$package is not a valid migration" >&2 # Direct output to stderr
        if [[ "$package" == *"migrator/migrations"* ]]; then
            echo "Ensure you're just providing the package name, not the full path" >&2
        fi
        exit 1
    fi
    mv "$MIGRATION_PATH" "$TEMP_DIR/$package"
    rm "$TEMP_DIR/$package/migration.go"
done

for package in $PACKAGES; do
    MIGRATION_NAME=$(echo "$package" | sed -E 's/^m_[0-9]+_to_m_[0-9]+_//')

    DESCRIPTION="$MIGRATION_NAME" make bootstrap_migration

    NEW_MIGRATION_PACKAGE=$(find migrator/migrations -name "*$MIGRATION_NAME" -type d | head -n1)

    if [[ -z "$NEW_MIGRATION_PACKAGE" ]] || ! [[ -d "$NEW_MIGRATION_PACKAGE" ]]; then
        echo "Failed to find newly created migration package for $MIGRATION_NAME" >&2
        exit 1
    fi

    rm -f "${NEW_MIGRATION_PACKAGE}"/{migration_impl,migration_test}.go

    mv "$TEMP_DIR/$package"/* "$NEW_MIGRATION_PACKAGE/"

    OLD_PREFIX="${package/$MIGRATION_NAME/}"
    OLD_PREFIX="${OLD_PREFIX//_/}"
    OLD_PREFIX="${OLD_PREFIX//\//}"

    NEW_PREFIX="${NEW_MIGRATION_PACKAGE/migrator\/migrations/}"
    NEW_PREFIX="${NEW_PREFIX/$MIGRATION_NAME/}"
    NEW_PREFIX="${NEW_PREFIX//[_\/]/}"

    if [[ "$OSTYPE" == "darwin"* ]]; then
        find "$NEW_MIGRATION_PACKAGE" -type f -exec sed -i '' "s/$OLD_PREFIX/$NEW_PREFIX/g" {} \;
    else
        find "$NEW_MIGRATION_PACKAGE" -type f -exec sed -i "s/$OLD_PREFIX/$NEW_PREFIX/g" {} \;
    fi
done
rm -rf "$TEMP_DIR"
