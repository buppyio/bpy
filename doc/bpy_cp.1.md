% bpy_cp(1)
% Andrew Chambers
% 2016

# NAME

bpy cp - copy a file or folder

# SYNOPSIS

The cp command lets you copy a file or folder from one path to another within a ref

# Usage

```bpy cp [-ref=default] src dest```

# Example

Copy a file

```
$ bpy ls
hello.txt
$ bpy cp hello.txt renamed.txt
$ bpy ls
hello.txt
renamed.txt
```

# SEE ALSO

**bpy(1)**
