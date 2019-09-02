#!/bin/sh

while true; do

  printf "READY\n";

  read line

  # Get EVENT_NAME and PAYLOAD_LENGTH
  for keypair in $line; do
    key="$(echo "$keypair" | cut -d ":" -f1)"
    value="$(echo "$keypair" | cut -d ":" -f2)"
    if [ "$key" == "len" ]; then
      PAYLOAD_LENGTH=$(($value))
    fi
    if [ "$key" == "eventname" ]; then
      EVENT_NAME=$value
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
    if [ "$key" == "expected" ]; then
      # 0 if the exit code was unexpected.
      # 1 if the exit code was expected.
      EXPECTED_EXIT_CODE=$value
    fi
  done

  printf "RESULT 2\nOK";

  # SIGKILL if:
  # - EVENT_NAME is PROCESS_STATE_FATAL
  # - EXPECTED_EXIT_CODE is unexpected
  # SIGQUIT if:
  # - Anything other than init process exited
  # - INIT_EXIT env var is set
  if [[ "$EVENT_NAME" == "PROCESS_STATE_FATAL" || "$EXPECTED_EXIT_CODE" == "0" ]]; then
    killall -SIGKILL supervisord
  elif [[ "$PROCESS_NAME" != "init" || -n "$INIT_EXIT" ]]; then
    killall -SIGQUIT supervisord
  fi

done