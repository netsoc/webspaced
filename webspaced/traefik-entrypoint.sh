#!/bin/sh
set -e

# TODO: is this necessary? traefik retries but it doesn't seem to pickup on changes then :(
until redis-cli ping; do
    sleep 0.1
done
#redis-cli set traefik yes

exec traefik "--providers.redis.endpoints=$REDIS" "$@"
