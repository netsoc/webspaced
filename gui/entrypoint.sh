#!/bin/sh
export PYTHONUNBUFFERED=1

if [ -n "$DEBUG" ]; then
    FLASK_ENV=development exec python -m webspace_ng serve
else
    exec gunicorn 'webspace_ng:app' \
        --bind '[::]:80' \
        --workers "$WORKERS" \
        --worker-class gevent
fi
