# aiware_qed


1. models : Contains websocket connection, serialization stuff


2. server:  start the ws endpoint `getwork` - start 2 servers by

```graphql
cd server
go build
./server -addr localhost:8080
./server -addr localhost:8090
```

3. client: client prog to connectto two servers, publish 5 times, then stop

```graphql
cd client
make build
./wss_client
```

you can play around with N clients...

Can stop the server, then client will attempt to connect within 1m

if client stop, server clear up

