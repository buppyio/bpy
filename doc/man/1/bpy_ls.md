% bpy_ls(1)
% Andrew Chambers
% 2016

# Name

bpy ls - list directory contents

# Synopsis

The ls command lets you list the contents of a bpy directory in a similar
way that the system ls command does on unixy systems.

# Usage

```bpy ls [-when=TIMESPEC] [path]```

# Example

List a directory

```
$ bpy ls /hello/
hello.txt
$ bpy ls
hello/
```

# See Also

**bpy(1)**, **bpy_timespec(7)**
