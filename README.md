# Minecraft RCON Console

```
go run ./cmd/mc-admin/main.go \
    --host <IP_ADDRESS> \
    --port 25575 \
    --password your_password
```

Default port is 25575, you can change it in `server.properties` file of your Minecraft server.
Default host is 'localhost', no need to specify it if you run the command on the same machine as the Minecraft server.
Password is required, you can set it in `server.properties` file of your Minecraft server.
