% bpy_hist(1)
% Andrew Chambers
% 2016

# NAME

bpy hist - list or prune a drive history

# SYNOPSIS

Whenever you make a change to your bpy files, data not removed. Bpy maintains a full history
of changes in case you ever need to revert a mistake such as accidental deletion of a file.
The bpy hist command allows you to list this history, and even prune it. Pruning your history 
in conjunction with running bpy_gc(1) is the only way to purge data from the pack file storage.

# Usage

```$ bpy hist list ```
```$ bpy hist prune [-all] [-older-than=TIMESPEC]```

# Example

View your history:

```
$ bpy hist list
```

Clear all history:

```
$ bpy hist prune -all
```

Clear history older than midday on the first of February 2016:

```
$ bpy hist prune -older-than="12:00:00 1/2/2016"
```

# SEE ALSO

**bpy(1)**, **bpy_timespec(7)**
