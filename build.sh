#!/usr/bin/env bash
set -euo pipefail

# Static binaries: os/user would otherwise link the build host's libc on Linux.
export CGO_ENABLED=0

rm -rf ./build

# build Linux
echo building Linux...
env GOOS=linux GOARCH=amd64 go build -trimpath -o ./build/linux_amd64/dynip-linux-amd64
env GOOS=linux GOARCH=arm64 go build -trimpath -o ./build/linux_arm64/dynip-linux-arm64

# build Windows
echo building Windows...
env GOOS=windows GOARCH=amd64 go build -trimpath -o ./build/windows_amd64/dynip-windows-amd64.exe

# build OSX
echo building OSX...
env GOOS=darwin GOARCH=amd64 go build -trimpath -o ./build/osx_amd64/dynip-osx-amd64
env GOOS=darwin GOARCH=arm64 go build -trimpath -o ./build/osx_arm64/dynip-osx-arm64

# build OpenBSD
echo building OpenBSD...
env GOOS=openbsd GOARCH=amd64 go build -trimpath -o ./build/openbsd_amd64/dynip-openbsd-amd64
