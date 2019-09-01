#!/bin/sh

while true; do

  printf "READY\n";

  read line

  # Get PAYLOAD_LENGTH
  for keypair in $line; do
    key="$(echo "$keypair" | cut -d ":" -f1)"
    value="$(echo "$keypair" | cut -d ":" -f2)"
    if [ "$key" == "len" ]; then
      PAYLOAD_LENGTH=$(($value))
    fi
  done

  # Get PAYLOAD
  PAYLOAD=$(head -c $PAYLOAD_LENGTH /dev/stdin);

  # Get PROCESS_NAME
  for keypair in $PAYLOAD; do
    key="$(echo "$keypair" | cut -d ":" -f1)"
    value="$(echo "$keypair" | cut -d ":" -f2)"
    if [ "$key" == "processname" ]; then
      PROCESS_NAME=$value
    fi
  done

  printf "RESULT 2\nOK";

  # Send SIGQUIT if any of the following is true:
  # - Anything other than init process exited
  # - INIT_EXIT env var is set
  if [[ "$PROCESS_NAME" != "init" || -n "$INIT_EXIT" ]]; then
    killall -SIGQUIT supervisord
  fi

done