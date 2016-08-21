% bpy_cat(1)
% Andrew Chambers
% 2016

# NAME

bpy cat - print the contents of a list of files to stdout

# SYNOPSIS

cat print the contents of a list of files to stdout, concatinating them into a single
stream. Useful for quickly inspecting the contents of a single file.

# Usage

```bpy cat [-ref=default] path```

# Example

Read the contents of a file

```
$ bpy ls
hello.txt
world.txt
$ bpy cat hello.txt world.txt
hello
world!
```

# SEE ALSO

**bpy(1)**
