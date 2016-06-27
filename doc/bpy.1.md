% bpy(1)
% Andrew Chambers
% 2016


# NAME

bpy - Buppy file storage.

# SYNOPSIS

``bpy`` is a tool for storing files securely on a remote server.
Data is encrypted on the client without giving the server access
to any of the decryption keys. This means the server cannot access
any of the stored data, and any tampering will be detected by the 
client using cryptographic signatures.

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
export BUPPY_REMOTE="ssh://yourserver/home/youruser/bpydata"
```

Finally, store a backup and tag it with a human readable tag/name.

```
echo "important document" > document.txt
bpy put -tag="$(PWD) $(DATE)" .
```

# SEE ALSO

**bpy_file_formats(7)**