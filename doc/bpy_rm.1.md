% bpy_rm(1)
% Andrew Chambers
% 2016

# NAME

bpy rm - remove a filr or folder from a ref

# SYNOPSIS

This command removes a file or folder from the given ref. The file contents and disk space
will not be reclaimed until the next garbage collection bpy_gc(1).

# Usage

```bpy put [-ref=default] path```

# Example

Remove a file

```
$ bpy ls
stuff/
$ bpy rm stuff
$ bpy ls
```

# SEE ALSO

**bpy(1)**
