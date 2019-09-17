# build Linux
echo building Linux...
env GOOS=linux GOARCH=amd64 go build -o ./build/linux_amd64/dynip-linux-amd64

# build Windows
echo building Windows...
env GOOS=windows GOARCH=amd64 go build -o ./build/windows_amd64/dynip-windows-amd64.exe

# build OSX
echo building OSX...
env GOOS=darwin GOARCH=amd64 go build -o ./build/osx_amd64/dynip-osx-amd64

# build OpenBSD
echo building OpenBSD...
env GOOS=openbsd GOARCH=amd64 go build -o ./build/openbsd_amd64/dynip-openbsd-amd64