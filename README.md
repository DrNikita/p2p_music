# Initial idea of the project
Create p2p application for listening to music

## Description
#### Song CIDs store in Global Playlist, which peers get while connecting to the network and update through gossipsub

## Next
- TODO: GlobalPlaylist --[move to]--> BoltDB
- FIX: func: : watch TODO note

#### Features completed
- Songs catalog storage
- Song promotion
- Song search
- Song streaming

# Network sturtup
- to start bootstrap node
```bash
go run .
```

- to start other nodes which will connect to bootstrap and other nodes in the network (copy bootstrap node ID in terminal):
```bash
go run . -dicovery <bootstrap node ID>
```



### Some notes
- UDP Buffer Sizes warning:
```bash
sudo sysctl -w net.core.rmem_max=7500000
sudo sysctl -w net.core.wmem_max=7500000
```