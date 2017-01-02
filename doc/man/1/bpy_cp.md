% bpy_cp(1)
% Andrew Chambers
% 2016

# Name

bpy cp - copy a file or folder

# Synopsis

The cp command lets you copy a file or folder from one path to another.

# Usage

```bpy cp [-at=TIMESPEC] src dest```

# Example

Copy a file:

```
$ bpy ls
hello.txt
$ bpy cp hello.txt renamed.txt
$ bpy ls
hello.txt
renamed.txt
```

# See Also

**bpy(1)**, **bpy_timespec(7)**
