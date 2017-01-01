% bpy_hist(1)
% Andrew Chambers
% 2016

# Name

bpy hist - list drive history

# Synopsis

The bpy hist command allows you to list this history.

Whenever you make a change to your bpy files, data not removed. Bpy maintains a full history
of changes between each garbage collection in case you ever need to revert a mistake such as accidental
deletion of a file.


# Usage

```$ bpy hist```

# Example

View your history:

```
$ bpy hist
```

# See Also

**bpy(1)**, **bpy_gc(1)**, **bpy_timespec(7)**
