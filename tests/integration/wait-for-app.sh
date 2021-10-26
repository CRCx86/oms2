#!/bin/sh

set -e

host="$1"
port="$2"
readiness_path="$3"
shift
shift
shift
cmd="$@"

full_url="http://$host:$port/$readiness_path"

until curl --output /dev/null --silent --fail  "$full_url"; do
  sleep 1
done

exec $cmd