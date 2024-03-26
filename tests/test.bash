#!/bin/bash

# Define the base URL and the custom auth token
baseUrl="https://utk-auth-go-production.up.railway.app"
authToken="9288F64872458E33152931BE497B1"

# Number of concurrent requests
requestCount=50

for i in $(seq 1 $requestCount); do
  # Generating random user-discord-id and guild-discord-id for each request
  userDiscordId=$((RANDOM))
  guildDiscordId=$((RANDOM))

  # The curl command
  curl -X POST "$baseUrl/generate-user-token?user-discord-id=$userDiscordId&guild-discord-id=$guildDiscordId" \
       -H "X-Custom-Auth: $authToken" \
       -H "Content-Type: application/json" >> out.txt &
        

  # Optional delay between requests
  # sleep 1
done

# Wait for all background jobs to finish
wait

echo "All requests sent."

