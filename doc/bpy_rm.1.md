% bpy_rm(1)
% Andrew Chambers
% 2016

# NAME

bpy rm - remove a file or folder

# SYNOPSIS

rm removes a file or folder. The file contents and disk space
will not be reclaimed until the next history clearing and garbage collection with bpy_hist(1) and bpy_gc(1).

# Usage

```$ bpy rm file1 [file2..]```

# Example

Remove a folder:

```
$ bpy ls
stuff/
$ bpy rm stuff
$ bpy ls
```

# SEE ALSO

**bpy(1)**
