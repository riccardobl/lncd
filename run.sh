#!/bin/bash
script_dir=$(dirname $0)
cd "$script_dir/snlncreceiver"
go run .
