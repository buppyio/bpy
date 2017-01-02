% bpy_rm(1)
% Andrew Chambers
% 2016

# Name

bpy rm - remove a file or folder.

# Synopsis

rm removes a file or folder from the buppy drive.
Remote space will not be reclaimed until the next garbage collection by bpy_gc(1).
If you made a mistake, bpy_revert(1) can be used for rolling back changes that were made
before the last garbage collection.

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

# See Also

**bpy(1)**, **bpy_gc(1)**, **bpy_revert(1)**
