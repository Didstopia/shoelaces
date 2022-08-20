#!/bin/sh

## TODO: Replace this with a Go app that does the same,
##       as this way we can continue to use the scratch base image

set -e
set -o pipefail

set -x

# Initialize the data volume if it is empty
if [[ ! -d "/data" || ! "$(ls -A /data)" ]]; then
  echo "Initializing data volume ..."
  ls -lah /shoelaces_default/data
  # cp -fr /shoelaces_default/data/mappings.yaml /data/mappings.yaml
  cp -fr /shoelaces_default/data/* /data/
  ls -lah /data
else
  echo "Data volume already initialized, skipping ..."
fi

# FIXME: This should be a part of the base image + entrypoint logic, not a dumb mkdir command!
# Always ensure that the env_overrides folder exists
mkdir -p /data/env_overrides

# Initialize the web volume if it is empty
if [[ ! -d "/web" || ! "$(ls -A /web)" ]]; then
  echo "Initializing web volume ..."
  ls -lah /shoelaces_default/web
  cp -fr /shoelaces_default/web/* /web/
  # cp -fr /shoelaces_default/web/ /web
  ls -lah /web
else
  echo "Web volume already initialized, skipping ..."
fi

echo "Fixing permissions ..."
chown -R ${PUID}:${PGID} /data /web

echo "Starting Shoelaces ..."
exec "$@"
