% bpy_ref(5)
% Andrew Chambers
% 2017

# Name

ref - The ref format used by bpy to store a linked list of drive history.

# Synopsis

Buppy store all history as a singly linked list of refs. Every time you made a change to a buppy
drive, it appends a new ref to the history chain. bpy_hist(1) allows you to print the entire ref chain.

Each ref is a binary structure containing the unix time of the change, the a hash pointing to a bpy_fs(5) and
a hash pointing to the previous ref in the hsitory chain.

Example refs as they appear on disk:
```
Ref with a previous node.
+------------------+
| CreatedAt[8]     |
| FSRoot[32]       |
| PreviousRef[32]  |
+------------------+
Ref without a previous node:
+------------------+
| CreatedAt[8]     |
| FSRoot[32]       |
+------------------+
```

# See Also

**bpy_hist(5)**, **bpy_fs(7)**
