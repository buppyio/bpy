% bpy_tar(1)
% Andrew Chambers
% 2016

# Name

bpy tar - export a bpy folder to a tar archive

# Synopsis

The tar command creates a tar archive out of the specified folder writing the output to stdout.

# Usage

```bpy tar [-at=TIMESPEC] src```

# Example

Fetch the entire drive as a tar file:

```
$ bpy tar / > out.tar
```

Fetch a specific folder as a tar file and add compression

```
$ bpy tar path/to/files | gzip > out.tar.gz
```

# See Also

**bpy(1)**
