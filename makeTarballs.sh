#!/bin/bash
rm -rf tmp
mkdir tmp
cp kdep tmp/
cp kdep-merge-inherited-values tmp/
for platform in linux darwin; do
	for arch in amd64; do
		for filename in base64encode base64decode sha256sum; do
			GOOS=$platform GOARCH=$arch CGO_ENABLED=0 go build -o tmp/kdep-$filename $filename.go
		done
		cd tmp
		tar -czvf ../$platform.tar.gz *
		cd ..
	done
done
