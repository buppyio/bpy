% bpy_env(7)
% Andrew Chambers
% 2016

# NAME

bpy env - environment variables

# SYNOPSIS

This page describes the various environment variables used by bpy(1).

# Environment Variables

## BPY_PATH

BPY_PATH defaults ```~/.bpy``` and is the path to where bpy reads its key file from, 
and also where all remote pack indexes are cached. This cache contains a record of file
indexes allowing bpy to locate data within the remote pack files. The cache also enables
bpy to keep track of what data it does not need to send to the server. It is safe to remove everthing inside the cache folder without losing data, because the
pack indexes are also stored on the remote and are redownloaded if needed.

The following is an example $BPY_PATH folder containing a bpy.key file, and a cache containing 
pack file indexes for the bpy key with the truncated key id ```111e0d...ba0dc8```.

```
~/.bpy
├── bpy.key
└── cache
    └── 111e0d...946ba0
        ├── ba8a15...e6ee2e.ebpack.index
        └── da1078...895c2f.ebpack.index
```

## BPY_REMOTE_CMD

$BPY_REMOTE_CMD is executed by the bpy command when it needs to establish a connection to
an instance of the bpy_remote(1) command. Stdin and stdout of this command must be piped to and from an instance
of the remote server.

Example:

```
$ export BPY_REMOTE_CMD="bpy remote /home/localuser/bpy_datadir"
$ export BPY_REMOTE_CMD="ssh $SERVER /bin/bpy remote /bpy_datadir"
```


# SEE ALSO

**bpy(1)**
