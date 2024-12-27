#!/bin/bash
source build.sh
chmod +x ../dist/lncd
export LNCD_DEBUG="true"
export LNCD_TIMEOUT="1m"
export LNCD_STATS_INTERVAL="10s"
export LNCD_DEV_UNSAFE_LOG="true"
../dist/lncd