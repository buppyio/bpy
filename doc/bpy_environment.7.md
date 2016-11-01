% bpy_environ(7)
% Andrew Chambers
% 2016

# Name

bpy env - environment variables

# Synopsis

This page describes the various environment variables used by bpy(1). The values bpy(1) is
using can be inspected usign the bpy_env(1) command.

# Environment Variables

## BPY_REMOTE_CMD

BPY_REMOTE_CMD is executed by the bpy command when it needs to establish a connection to
an instance of the bpy_remote(1) command. Stdin and stdout of this command must be piped to and from an instance
of the remote server. It has no default value.

Example:

```
$ export BPY_REMOTE_CMD="bpy remote /home/localuser/bpy_datadir"
$ export BPY_REMOTE_CMD="ssh $SERVER /bin/bpy remote /bpy_datadir"
```

## BPY_PATH

BPY_PATH defaults to ```$HOME/.bpy``` and is the path that many other variables base their
default values off.

## BPY_ICACHE_PATH

BPY_ICACHE_PATH defaults to ```$BPY_PATH/icache/``` and is the directory containing the index cache. The index cache contains a record of file
key and value offsets allowing bpy to locate data within the remote pack files. The cache also enables
bpy to keep track of what data it does not need to send to the server. 
It is safe to remove everthing inside this cache folder without losing data, because the
pack indexes are also stored on the remote and are redownloaded if needed.

An example directory tree populated with two indexes:


```
$BPY_ICACHE_PATH
└── 111e0d...946ba0
    ├── ba8a15...e6ee2e.ebpack.index
    └── da1078...895c2f.ebpack.index
```

## BPY_CACHE_FILE

BPY_CACHE_FILE defaults to ```$BPY_PATH/chunks.db``` and is a boltdb database file containing cached data chunks.
This file is used to store data locally and greatly increase the performance of bpy(1) when accessing data.
It is safe to remove this file when bpy is not using it and the data will be refetched from the remote.

## BPY_CACHE_SIZE

BPY_CACHE_SIZE defaults to ```512 megabytes``` and is the maximum size of the data stored in the BPY_CACHE_FILE.
If the file contents exceed this size, the least recently used values will be discarded.

## BPY_CACHE_LISTEN_ADDR

BPY_CACHE_LISTEN_ADDR defaults to ```127.0.0.1:8877``` and is the address the bpy(1) will connect to for accessing the 
local data cache. If no service is listening on this address, bpy(1) will spawn a background instance of bpy_cache_daemon(1) using
the configuration from the current environment.

# SEE ALSO

**bpy(1)**, **bpy_env(1)**
