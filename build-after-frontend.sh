#!/bin/bash
# vite emptyOutDir wipes the go embed declaration and static data inside web/;
# restore them from git before compiling.
set -e
cd "$(dirname "$0")"
git checkout -- core/cmd/server/web/web.go
mkdir -p core/cmd/server/web/static
git checkout -- core/cmd/server/web/static/china.json core/cmd/server/web/static/world.json
echo "web.go + static restored"
