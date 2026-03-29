#!/usr/bin/env bash
# Convert your .pem file to a single-line escaped string
cat $1 | awk '{printf "%s\\n", $0}'
