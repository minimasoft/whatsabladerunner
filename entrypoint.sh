#!/bin/sh

mkdir -p /data/config /data/logs /data/plain_media /data/plain_media/image /data/plain_media/video /data/plain_media/audio /data/plain_media/docs

# If the config directory is empty, populate it with templates
if [ -z "$(ls -A /data/config)" ]; then
    echo "Initializing /data/config with templates..."
    cp -r /usr/local/share/whatsabladerunner/config-template/. /data/config/
fi

cd /data

exec /usr/local/bin/whatsabladerunner
