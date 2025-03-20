# Initial idea of the project
Create p2p application for listening to music

## Next steps plan
- create playlist topic to create access for all peers to all song CIDs
- share files
- find songs
---------------------------------------------------------------------------------------
- - - Check how to work with STREAMS
- store audiofile as a single file on disk, send it by parts, dividing this file into small parts(ffmpe for example)
  and store this file in-memory
- think about change-sender-mechanism (if current sender shut down) (md can connect to multiple peer, but read from one,
  then if something happened change sender) 
- if I will store audio frames in-memory one song will take for about 1GB => huynya
    - mb I can optimize this algorithm by rewriting this frames when then were sent
        - so in this case I think I can store a song as a single file for now
---------------------------------------------------------------------------------------
- use kdht.Provide() to mark peer as song provider
- use kdht.FindProviders() to find all song providers
- store song as small parts of the file
- store this small parts on disk of the peer
- store path to the song in NoSQL/SQL DB
- So, in this case I must provide an algorithm to divide a song to small pieces (ffmper can be used:)

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