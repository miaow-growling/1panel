#!/bin/bash
# vite emptyOutDir wipes the go embed declaration living inside web/; restore it.
set -e
cd "$(dirname "$0")"
git checkout -- core/cmd/server/web/web.go
echo "web.go restored"
