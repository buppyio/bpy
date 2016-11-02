% bpy_env(1)
% Andrew Chambers
% 2016

# Name

bpy env - print the bpy_environment(7)

# Synopsis

The env command lets you print the current bpy_environment(7) variables to allow troubleshooting.

# Usage

```bpy env```

# Example

Print the bpy environment

```
$ bpy env
BPY_REMOTE_CMD=ssh 189205894861979649@buppy.io bpy remote
BPY_PATH=/home/user/.bpy
BPY_ICACHE_PATH=/home/user/.bpy/icache
BPY_CACHE_FILE=/home/user/.bpy/chunks.db
BPY_CACHE_SIZE=536870912
BPY_CACHE_LISTEN_ADDR=127.0.0.1:8877
```

# SEE ALSO

**bpy(1)**, **bpy_environment(7)**
