set -e

npm i
./node_modules/.bin/gulp
dir2go -pkgname static -dir www -o files.go
