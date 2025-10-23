#!/bin/bash
set -e
for dir in all2avif all2jxl static2avif static2jxl dynamic2avif dynamic2jxl video2mov merge_xmp deduplicate_media universal_converter; do
  cd $dir
  go mod tidy
  go build -o bin/${dir//-/_} main.go
  cd ..
done
