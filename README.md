# Initial idea of the project
Create p2p application for listening to music

# Network sturtup:
- to start bootstrap node
```bash
go run .
```

- to start other nodes which will connect to bootstrap and other nodes in the network (copy bootstrap node ID in terminal):
```bash
go run . -dicovery <bootstrap node ID>
```



### Some notes:
- UDP Buffer Sizes warning:
```bash
sudo sysctl -w net.core.rmem_max=7500000
sudo sysctl -w net.core.wmem_max=7500000
```