% bpy_gc(1)
% Andrew Chambers
% 2016

# Name

bpy gc - clear history, garbage collect and reclaim unreachable space on the remote drive.

# Synopsis

gc means garbage collection, this command is how space is reclaimed after
files are rm'd using bpy_rm(1). During a garbage collection, the remote will block any updates, 
and any attempt to start a second collection will cause the original collection to safely fail.

The gc command works by starting from the root of the file system and its history and traversing the data, marking every chunk that is reachable.
After the marking phase is completed, the gc will perform what is known as a 'sweep'.
The sweep will traverse remote pack file indexes, fetching reachable data, and repacking it
in new pack files with the garbage removed. Old pack files are only deleted once
the new pack file is safely commited to disk storage, so disk usage may temporarily rise before
the collection is completed.

Because each pack file uses a unique IV key in its encryption, reachable data must be downloaded and reuploaded,
this process can take some time when there are large amounts of packfiles that have unreachable data. A collection can be
canceled at any time and resuming will not need to reprocess all the same data because repacked files will be fully reachable.

The possibly slow speed of GC is partially mitigated by the local bpy cache to completely remove
the overhead of data fetching.

# Usage

```$ bpy gc```

# Example

run the garbage collector

```
$ bpy gc
```

# SEE ALSO

**bpy(1)**, **bpy_hist(1)**, **bpy_undo(1)**
