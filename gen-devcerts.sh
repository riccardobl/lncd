#!/bin/bash
mkdir -p certs
openssl req -x509 -newkey rsa:2048 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes \
  -subj "/C=IT/ST=Venice/L=Venice/O=LNCD/OU=LNCD/CN=localhost"