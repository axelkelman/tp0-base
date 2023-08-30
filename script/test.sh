#!/usr/bin/env bash
source config.sh

echo "Sending message = $MESSAGE to the server"
response=$(echo -e "$MESSAGE\n" | nc $SERVER_IP $SERVER_PORT)
if [ "$response" == "$MESSAGE" ]; then
    echo "Server response is OK | response = $response"
else
    echo "Server response is not OK | response = $response"
fi
