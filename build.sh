#!/bin/bash
source buildvars.sh

script_dir=$(dirname $0)
cd "$script_dir/lncd"
mkdir -p ../dist
go build -o ../dist/lncd -tags="$RPC_TAGS" .
