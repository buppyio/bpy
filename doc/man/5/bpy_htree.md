% bpy_htree(5)
% Andrew Chambers
% 2016

# Name

htree - Hash tree format for storing streams of data.

# Synopsis

bpy stores all data internally as a hash tree data structure. Each node in tree is given an address which
reflects a bpack key used to locate the node data.
The node address is calculated as ```SHA256(DEFLATE(NODEDATA))```, where the node data stored at the address is either a
flate compressed list of stream offsets and child node addresses, or for leaf nodes, the flate compressed
stream data itself. 

The htree data structure is important to bpy as it enables the following properties:

- Large amounts of data can be stored in chunks that are small enough to not violate the size
  limits of bpy_bpack(5) files.
- Provide relatively efficient random access to the data stream while walking chunk contents,
  this allows 'seeking' when data streams are being accessed.
- Given two trees with identicle sub trees, all data can be cheaply deduplicated on disk by checking
  if the sub tree address is present in any bpy_ebpack(5) file.
- Provides compression for runs of values in data.

The following diagram shows what a 3 node htree will look like stored on disk in a bpack file:

```
Chunk0, address = SHA256(Deflate(Chunk0))
+------------------+
| Flate compressed |
| +--------------+ |
| | depth0[1]    | | depth0 = 1 
| | offset1[8]   | | offset1 = 0
| | address1[32] | | address1 = SHA256(Deflate(Chunk1))
| | offset2[8]   | | offset2 = N1
| | address1[32] | | address2 = SHA256(Deflate(Chunk2))
| +--------------+ |
+------------------+
Chunk1, address = SHA256(Deflate(Chunk2))
+------------------+
| Flate compressed |
| +--------------+ |
| | depth1[1]    | | depth1 = 0
| | data1[N1]    | |
| +--------------+ |
+------------------+
Chunk2, address = SHA256(Deflate(Chunk2))
+------------------+
| Flate compressed |
| +--------------+ |
| | depth2[1]    | | depth2 = 0
| | data2[N2]    | |
| +--------------+ |
+------------------+
```

# See Also

**bpy_bpack(5)**, **bpy_fs(7)**
