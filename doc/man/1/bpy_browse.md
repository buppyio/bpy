% bpy_browse(1)
% Andrew Chambers
% 2016

# Name

bpy browse - launch a webserver that allows browsing of your bpy files.

# Synopsis

The browse command allows a convenient web based user interface. 
When run with no arguments it starts a web server on the default port,
and automatically opens a web browser pointing to this URL so you may browse
your files.

# Usage

```bpy browse [-addr=127.0.0.1:8000] [-no-browser]```

provide the -no-browser flag to suppress the spawning of a web browser.

# Example

Run a local webserver where you can browse the bpy file system:

```
$ bpy browse
```

# See Also

**bpy(1)**
