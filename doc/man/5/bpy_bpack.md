% bpy_bpack(5)
% Andrew Chambers
% 2016

# Name

bpack - bpack is the file format used by bpy as an on disk mapping of keys to values.

# Synopsis

Bpy stores file data on disk as a collection of bpack files. Bpack files
are a simple format that stores a list of values concatenated, with an index
sorted by key, that can be read quickly without scanning the values inside the pack.

The format is designed so that bpy can write the pack files in a single
data stream without back tracking.

In typical usage, the keys used by bpy are all sha256 hashes, using the 
pack files as a content addressed storage system.

The following is an example diagram showing the layout of 3 values and the index as stored on disk:

```
+-------------+ <- off1
| val[N1]     |
+-------------+ <- off2
| val[N2]     |
+-------------+ <- off3
| val[N3]     |
+-------------+ <- idxstart
| keylen[3]   |
| key[keylen] |
| off1[8]     |
| N1[3]       |
+-------------+
| keylen[3]   |
| key[keylen] |
| off2[8]     |
| N2[3]       |
+-------------+
| keylen[3]   |
| key[keylen] |
| off3[8]     |
| N3[3]       |
+-------------+
| idxstart[8] |
+-------------+
```

# See Also

**bpy(1)**
