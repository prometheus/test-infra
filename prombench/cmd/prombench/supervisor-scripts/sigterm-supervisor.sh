#!/bin/sh

while true; do
  printf "READY\n";
  read line
  echo "Processing Event: $line" >&2;
  printf "RESULT 2\nOK";
  killall -SIGTERM supervisord
done