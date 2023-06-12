#!/bin/bash

URL=http://app:8080/log

for ((i=1; ; i++)); do
  log='{
    "id": '$i',
    "unix_ts": '$(date +%s)',
    "user_id": '$i',
    "event_name": "login"
  }'

  curl -X POST -H "Content-Type: application/json" -d "$log" $URL

  if ((i % 1000 == 0)); then
    echo "Sent $i logs"
  fi

  sleep 0.001
done
