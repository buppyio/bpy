% bpy_put(1)
% Andrew Chambers
% 2016

# NAME

bpy put - put a local folder into the bpy drive.

# SYNOPSIS

The put command lets you upload a local file or folder into the bpy root or a subfolder.

# Usage

```bpy put src [dest]```

# Example

Put the current working directory into the root of the bpy drive:

```
$ pwd
/home/ac/stuff
$ bpy put .
$ bpy ls
stuff/
```

Put a single file into the stuff folder:

```
$ bpy put /home/ac/somefile.txt stuff/
$ bpy ls
stuff/
$ bpy ls stuff
somefile.txt
```

Put a single file into the root directory with a different name:

```
$ bpy put /home/ac/somefile.txt foo.txt
$ bpy ls
foo.txt
```

# SEE ALSO

**bpy(1)**
