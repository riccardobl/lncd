#!/bin/bash
script_dir=$(dirname $0)
cd "$script_dir/lncd"
mkdir -p ../dist
go build -o ../dist/lncd .
