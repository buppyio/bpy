% bpy_browse(1)
% Andrew Chambers
% 2016

# NAME

bpy browse - launch a webserver that allows browsing

# SYNOPSIS

The browse command allows a convenient web based user interface that updates when
the remote data changes. 

# Usage

```bpy browse [-addr=127.0.0.1:8000] [-no-browser]```

provide the -no-browser flag to suppress the spawning of a web browser.

# Example

Run a local webserver where you can browse the bpy file system

```
$ bpy browse
```

# SEE ALSO

**bpy(1)**
