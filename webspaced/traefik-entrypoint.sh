#!/bin/sh
set -e

REDIS_HOST=${REDIS_HOST:-127.0.0.1}
REDIS_PORT=${REDIS_PORT:-6379}

redis() {
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" "$@" > /dev/null 2>&1
}

until redis ping; do
    sleep 0.1
done

# We need a dummy Traefik config so the configuration is never empty
redis set traefik/http/middlewares/dummy/redirectScheme/scheme 'https'

exec traefik "--providers.redis.endpoints=$REDIS_HOST:$REDIS_PORT" "$@"
