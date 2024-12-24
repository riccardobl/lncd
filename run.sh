#!/bin/bash
script_dir=$(dirname $0)
cd "$script_dir/lncd"
go run .
