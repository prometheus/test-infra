#!/bin/sh

printf "READY\n"; # required for Header information by supervisord

while read line; do
  echo "Processing Event: $line" >&2; # send to stderr
  killall -SIGTERM supervisord
done < /dev/stdin