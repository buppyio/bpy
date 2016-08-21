% bpy_tar(1)
% Andrew Chambers
% 2016

# NAME

bpy tar - export a bpy folder to a tar archive

# SYNOPSIS

The tar command creates a tar archive out of the specified ref and folder.

# Usage

```bpy tar [-ref=default] src```

# Example

Fetch the entire ref as a tar file

```
$ bpy tar / > out.tar
```

Fetch a specific folder as a tar file and add compression

```
$ bpy tar path/to/files | gzip > out.tar.gz
```

# SEE ALSO

**bpy(1)**
