% bpy_drive(7)
% Andrew Chambers
% 2016

# Name

drive - a directory containing encrypted bpy file data.

# Synopsis

A bpy drive is a directory on a machine that is interacted with via the bpy_remote(1) command.
It contains a database of metadata holding important internal values to the drive, and a folder
container bpy_ebpack(5) files, which contain the encrypted data chunks comprising the bpy filesystem.

# SEE ALSO

**bpy(1)**, **bpy_bpack(5)**, **bpy_ebpack(5)**