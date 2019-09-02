#!/bin/sh

# Why we need this script?
# This is a Supervisord listener script : http://supervisord.org/events.html
#
# Supervisord processes emit events, two such events are:
# PROCESS_STATE_EXITED and PROCESS_STATE_FATAL
#
# PROCESS_STATE_FATAL: The process exited from the RUNNING state (expectedly or unexpectedly).
# PROCESS_STATE_EXITED: The process could not be started successfully.
# more info on process states: http://supervisord.org/subprocess.html#process-states
#
# This script handles what should be done when a script emits these events.
#
# How this script works?
# Supervisord emits event info to stdin of the event listener
# and takes in listener updates on stdout. This script just reads from stdin
# and prints necessary supervisord related updtes to stdout and then handles the event.

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