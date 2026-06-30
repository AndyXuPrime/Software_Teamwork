#!/bin/bash
set -euo pipefail

PACKAGES=(
    "./internal/admin/..."
    "./internal/cli/..."
    "./internal/common/..."
    "./internal/dao/..."
    "./internal/engine/..."
    "./internal/entity/..."
    "./internal/handler/..."
    "./internal/ingestion/..."
    "./internal/router/..."
    "./internal/server/..."
    "./internal/service/..."
    "./internal/storage/..."
    "./internal/tokenizer/..."
    "./internal/utility/..."
)

echo "Running tests for available Go packages..."
for pkg in "${PACKAGES[@]}"; do
    package_dir="${pkg#./}"
    package_dir="${package_dir%/...}"
    if [ ! -d "$package_dir" ]; then
        echo "=== Skipping $pkg (missing $package_dir) ==="
        echo ""
        continue
    fi

    echo "=== Testing $pkg ==="
    go test "$pkg" -v -cover
    echo ""
done
