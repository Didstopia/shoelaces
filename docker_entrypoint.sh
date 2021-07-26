#!/bin/sh

## TODO: Replace this with a Go app that does the same,
##       as this way we can continue to use the scratch base image

set -e
set -o pipefail

set -x

if [ ! -d "/data" ]; then
  ls -lah /shoelaces_default/data
  cp -fr /shoelaces_default/data/mappings.yaml /data/mappings.yaml
  ls -lah /data
fi

if [ ! -d "/web" ]; then
  ls -lah /shoelaces_default/web
  cp -fr /shoelaces_default/web/* /web/
  ls -lah /web
fi

exec "$@"
