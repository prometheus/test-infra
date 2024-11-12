#!/bin/sh

KEY_FILE="/key/key.yml"

if [[ -f "$KEY_FILE" ]]; then
    echo "Found key.yml, proceeding with download..."
    /bin/block-sync download --tsdb-path=/data --objstore.config-file=/config/object-config.yml --key=$KEY_FILE
else
    echo "key.yml not found, skipping download."
    exit 0
fi