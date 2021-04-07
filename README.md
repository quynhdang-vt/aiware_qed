# aiware_qed


1. models : Contains websocket connection, serialization stuff


2. server:  start the ws endpoint `getwork` - start 2 servers by

```graphql
cd server
make build test1
```

Then in another terminal

```graphql
make test2
```

3. client: client prog to connectto two servers, publish 5 times, then stop

```graphql
cd client
make build
make test
```

you can play around with N clients...

Can stop the server, then client will attempt to connect within 1m

if client stop, server clear up

