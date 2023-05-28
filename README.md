# Golang Raft NBA Application

## What's in here
Raft protocol is set to serve in the range of 7010, 7011 onwards.
The Raft API for handling Nodes and Status plus NBA Stats is set to serve in the range of 7110, 7210 onwards
We are going to start 3 nodes and join them together.
Once a node joins, it gets synced with all the data.


# How to test it
1.Start the bootstrapping node.
You need to bootstrap a single node, once. For that, set the BOOTSTRAP_CLUSTER=true environment variable once.
Each server needs to bind a port. use the PORT=int environment variable.
```
BOOTSTRAP_CLUSTER=true PORT=7010 go run cmd/main.go
```

2.Start another "regular" node.
```
PORT=7011 go run cmd/main.go
```

3.Join them together by sending a join command to the leader (and only the leader obviously):
```
curl --location --request POST 'http://localhost:7110/raft/join' \
--header 'Content-Type: application/json' \
--data-raw '{"id": "7011", "address": "localhost:7011"}'
```

4.Publish an NBA stat event:
```
curl --location --request POST 'http://localhost:7110/nba/stats' \
--header 'Content-Type: application/json' \
--data-raw '{"player_id": "1", "player_name": "dubi_gal", "game_time": "10:52", "stat": "personal_foul"}'
```
Then inspect both node's logs to see how that data is replicated

5.start the 3rd node, join it, and check the logs to see how the data is being replicated
```
PORT=7012 go run cmd/main.go
curl --location --request POST 'http://localhost:7110/raft/join' \
--header 'Content-Type: application/json' \
--data-raw '{"id": "7012", "address": "localhost:7012"}'
```

6.from here on, you can replace the leader node and see how the nodes elect a new leader and replicate the new node.
  Once the leader is alive, commands will have to be submitted to the new leader. sending a command to the wrong node
  will result in an error, indicating what's the address and id of the newly elected leader.




## Commands
### Join Node
The following command will join the 2nd and 3rd nodes.
```
curl --location --request POST 'http://localhost:7110/raft/join' \
--header 'Content-Type: application/json' \
--data-raw '{"id": "7011", "address": "localhost:7011"}'
```

### Remove Node
```
curl --location --request POST 'http://localhost:7110/raft/remove' \
--header 'Content-Type: application/json' \
--data-raw '{"id": "7011", "address": "localhost:7011"}'
```

### Get Cluster Status
```
curl --location --request GET 'http://localhost:7110/raft/nodes' \
--header 'Content-Type: application/json'
```

### Add NBA Stat
```
curl --location --request POST 'http://localhost:7110/nba/stats' \
--header 'Content-Type: application/json' \
--data-raw '{"player_id": "1", "player_name": "dubi_gal", "game_time": "10:52", "stat": "personal_foul"}'
```
