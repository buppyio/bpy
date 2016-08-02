% bpy(1)
% Andrew Chambers
% 2016

# NAME

bpy - Buppy file storage from https://buppy.io

# SYNOPSIS

``bpy`` is a tool for storing files securely on a remote server.
Data is encrypted and deduplicated by the client without giving the server access
to any of the decryption keys. This means the server cannot access
any of the stored data, and any data tampering will be detected by the 
client.

# Getting started

To use bpy you need to generate an encryption key. This key must be
kept secret. N.B. Loss of this key means all data stored using
the key will be lost *FOREVER*.

```
bpy new-key -f ~/.bpy/bpy.key
```

Next setup the remote server data is going to be stored in. It requires
the bpy binary installed on your server, and passwordless ssh access to your
server.

```
export BPY_REMOTE="ssh://yourserver/home/youruser/bpydata"
```

Finally, store a backup

```
echo "important document" > document.txt
bpy put .
```

Once you have data stored in your bpy drive, there are multiple ways to retrieve it, try any
of the follwing examples.

View your data with 'ls' and read the contents with 'cat':

```
bpy ls
bpy cat document.txt
```

View your data via the web interface:

```
bpy browse
```

Serve your drive as a 9p network file system:

```
bpy 9p
```

and others...

# SEE ALSO

**bpy_commands(1)** **bpy_file_formats(7)**
