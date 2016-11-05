% bpy_fs(5)
% Andrew Chambers
% 2016

# Name

fs - The bpy file system

# Synopsis

Bpy stores files in a virtual file system built on top of a collection of bpy_ebpack(5) files.
Every directory in the file system itself is comprised of bpy_htree(5) data structure, which
has a data stream describing the directory contents.

The directory stream data contains a list of directory entries, with the first entry
containing a special entry with name '.' that refers to the current directory. The
'Data' field for the '.' entry is an unspecified value because hash trees cannot contain
circular references.

The directory entries are serialized as the following structure:

```
+---------------+
| NameLen[2]    | Little endian length of directory entry.
+---------------+
| Name[NameLen] | Directory entry name.
+---------------+
| Size[8]       | Little endian size of the entry, 0 for directories.
+---------------+
| Mode[4]       | Little endian file flags, described [https://golang.org/pkg/os/#FileMode ](here).
+---------------+
| ModTime[8]    | Little endian modification time.
+---------------+
| DataDepth[1]  | The height of contents htree.
+---------------+
| Data[32]      | The hash pointing to the sub htree containing the file/directory contents.
+---------------+
```

# SEE ALSO

**bpy(1)**, **bpy_ebpack(5)**, **bpy_htree(5)**
