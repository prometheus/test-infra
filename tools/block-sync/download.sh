#!/bin/sh

PATH_FILE="/key/bucket-config.yml"

if [[ -f "$PATH_FILE" ]]; then
    echo "Found bucket-config.yml, proceeding with download..."
    /bin/block-sync download --tsdb-path=/data --objstore.config-file=/config/object-config.yml --path=$PATH_FILE
else
    echo "bucket-config.yml not found, skipping download."
    exit 0
fi
