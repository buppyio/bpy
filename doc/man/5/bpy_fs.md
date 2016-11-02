% bpy_fs(5)
% Andrew Chambers
% 2016

# Name

fs - The bpy file system

# Synopsis

Bpy stores files in a virtual file system built on top of a collection of bpy_ebpack(5) files.
Every directory in the file system itself is comprised of bpy_htree(5) data structure, which
has a data stream describing the directory contents.

The directory data is laid out as follows as follows:

XXX



# SEE ALSO

**bpy(1)**
