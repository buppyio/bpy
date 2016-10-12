% bpy_timespec(7)
% Andrew Chambers
% 2016

# NAME

bpy time spec

# SYNOPSIS

This page describes the time spec format used by the majority of bpy(1) commands.

A time spec can take one of two formats, an exact date, or a relative time spec.
Exact dates are specified in the format "hh:mm:ss dd/MM/YYYY" while relative times
are specified as "NN(s|m|h) ago". 

# Example

```
bpy ls -when="5m ago" # 5 minutes ago
bpy ls -when="12:00:00 1/2/2016"  # midday on the first of February 2016
```