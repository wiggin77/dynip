# build Linux
echo building Linux...
env GOOS=linux GOARCH=amd64 go build -o ./build/linux_amd64/dynip

# build Windows
echo building Windows...
env GOOS=windows GOARCH=amd64 go build -o ./build/windows_amd64/dynip.exe

# build OSX
echo building OSX...
env GOOS=darwin GOARCH=amd64 go build -o ./build/osx_amd64/dynip