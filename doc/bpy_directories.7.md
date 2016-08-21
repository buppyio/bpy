% bpy_directories(7)
% Andrew Chambers
% 2016

# NAME

bpy directories

# SYNOPSIS

This page describes the various directories that bpy uses on both the remote
end, and on the client side.

# $BPY_PATH

BPY_PATH defaults ```~/.bpy``` and is the path to where bpy reads its key file from, 
and also where all remote pack indexes are cached. This cache contains a record of file
indexes allowing bpy to locate data within the remote pack files. The cache also enables
bpy to keep track of what data it does not need to send to the server. It is safe to remove everthing inside the cache folder without losing data, because the
pack indexes are also stored on the remote and are redownloaded if needed.

The following is an example $BPY_PATH folder containing a bpy.key file, and a cache containing 
pack file indexes for the bpy key with the key id ```111e0d8ce5f3dd74d344c751995652d43f083beb40019ccc571be89946ba0dc8```.

```
~/.bpy
├── bpy.key
└── cache
    └── 111e0d8ce5f3dd74d344c751995652d43f083beb40019ccc571be89946ba0dc8
        ├── ba8a15d092c43f1f8a210728f17fc2bbb801448b488aa6b831701a2d4ae6ee2e.ebpack.index
        └── da1078455832c4f848557ddb4a5774003a0ae7537a74d061d83f194386895c2f.ebpack.index
```
