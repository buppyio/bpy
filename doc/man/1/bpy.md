% bpy(1)
% Andrew Chambers
% 2016

# Name

bpy - Buppy file storage from https://buppy.io

# Synopsis

``bpy`` is a tool for storing files securely on the buppy.io servers.
Data is encrypted and deduplicated by the client without giving the server access
to any of the decryption keys. This means the server cannot access
any of the stored data, and any data tampering will be detected by the 
client.

# Getting started

First create an account at https://buppy.io and authorize ssh public key on your
buppy drive.

To use bpy you need to generate an encryption key. This key must be
kept secret. N.B. Loss of this key means all data stored using
the key will be lost *FOREVER*.

```
bpy new-key
```

Next setup the command bpy runs to connect to the remote server.
This command is a way to establish an ssh connection to the buppy.io server, 
Here we use openssh where DRIVEID is the value fetched from the buppy.io drive dashboard:

```
export BPY_REMOTE_CMD="ssh bpy@buppy.io drive $DRIVEID"
```

Finally, we can store a file in our buppy drive

```
echo "important document" > document.txt
bpy put document.txt
```

Once you have data stored in your bpy drive, there are multiple ways to retrieve it, try any
of the follwing examples.

View your data via the web interface:

```
bpy browse
```

View your data with 'ls' and read the contents with 'cat':

```
bpy ls
bpy cat document.txt
```

# Sub Commands

## browse
Launch a webserver and browse files a web browser.

## cat
Read the contents of one or more file.

## chmod
Change the permissions of a file or folder in the specifed ref.

## cp
Copy a file or folder.

## gc
Run the garbage collector to reclaim unused space and merge small pack files.

## get
Download the contents of a folder.

## hist
Fetch the drive history for recovery of rm'd files.

## ls
Get a directory listing of the specified folder.

## mkdir
Make a new 

## mv
Move a file or folder.

## new-key
Generate a local key file used by bpy for encrypting data.

## put
Upload a local folder or file.

## rm
Remove a file or folder.

## tar
Create a tar archive from the contents of the specified folder.

## zip
Create a zip archive from the contents of the specified folder.

## 9p
Launch a 9p server that can be used for mounting the bpy drive as a read only filesystem on platforms that support
it.

# SEE ALSO

**bpy_browse(1)**, **bpy_env(1)**, **bpy_hist(1)**, **bpy_mkdir(1)**, **bpy_rm(1)**,
**bpy_cat(1)**, **bpy_gc(1)**, **bpy_ls(1)**, **bpy_mv(1)**, **bpy_tar(1)**,
**bpy_cp(1)**, **bpy_get(1)**, **bpy(1)**, **bpy_put(1)**, **bpy_zip(1)**, **bpy_environment(7)**
