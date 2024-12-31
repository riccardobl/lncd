#!/bin/bash
source build.sh
chmod +x dist/lncd
cd dist

export LNCD_DEBUG="true"
export LNCD_TIMEOUT="1m"
export LNCD_STATS_INTERVAL="10s"
export LNCD_DEV_UNSAFE_LOG="true"

if [ -f ../certs/cert.pem ]; then
  export LNCD_TLS_CERT_PATH="../certs/cert.pem"
  export LNCD_TLS_KEY_PATH="../certs/key.pem"
fi

./lncd