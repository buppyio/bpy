% bpy_revert(1)
% Andrew Chambers
% 2016

# Name

bpy revert - revert a change

# Synopsis

The revert command allows your to restore your bpy drive to any entry in the current history.

# Usage

```$ bpy revert $HASH```

# Example

Remove a file, and then revert the change:

```
$ bpy ls
folder/
$ bpy rm folder
$ bpy ls
$ bpy hist
6dfe5f6e6a89861f622106ce810b3f93b48eac6b5482fa2d1797839eaa0c79a8 ...
9106ce424181e5840d4d0fb4e96c42e446a61f056874bc224cbc29c2ac185f3a ...
$ bpy revert 9106ce424181e5840d4d0fb4e96c42e446a61f056874bc224cbc29c2ac185f3a
$ bpy ls
folder/
```

# See Also

**bpy(1)**, **bpy_hist(1)**
