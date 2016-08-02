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
bpy 9p -addr=127.0.0.1:9001
```
and then in another terminal mount the 9p file system (linux example)
```
sudo mount -t 9p -o port=9001 127.0.0.1 /mnt
```

# Sub Commands

## browse
Launch a webserver and browse a bpy ref via a web browser

## cat
Read the contents of one or more file

## chmod
Change the permissions of a file or folder in the specifed ref

## cp
Copy a file or folder and update the specified ref

## get
Download the contents of a bpy ref folder

## get
Download the contents of a bpy ref folder

## history
Fetch hash history of a given ref

## ls
Get a directory listing of the specified ref and folder

## mv
Move a file or folder and update the specified ref

## new-key
Generate a local key file used by bpy for encrypting data

## put
Upload a local folder or file into the specifed ref

## repack
Merge small remote packs to increase performance

## rm
Remove a file or folder from the specified ref

## tag
Make a new named ref based on the current value of an existing ref

## tar
Create a tar archive from the contents of the specified ref and folder

## zip
Create a zip archive from the contents of the specified ref and folder

## 9p
Launch a 9p server and serve the given ref as a 9p filesystem


# SEE ALSO

**bpy_directories(7)** **bpy_env_variables(7)** **bpy_file_formats(7)**
