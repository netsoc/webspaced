#!/bin/sh
set -e

REDIS_HOST=${REDIS_HOST:-127.0.0.1}
REDIS_PORT=${REDIS_PORT:-6379}

until redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping; do
    sleep 0.1
done
redis-cli set traefik yes

exec traefik "--providers.redis.endpoints=$REDIS_HOST:$REDIS_PORT" "$@"
