#!/bin/bash
script_dir=$(dirname $0)
cd "$script_dir/snlncreceiver"
mkdir -p ../dist
go build -o ../dist/snlncreceiver .
