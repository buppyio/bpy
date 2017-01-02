% bpy_key(5)
% Andrew Chambers
% 2016

# Name

key - key files contain the bpy encryption/decryption keys

# Synopsis

The bpy(1) client uses a keyfile during normal operation to ensure data integrity and privacy.
The key file itself is a json file encoding three seperate values described as follows.

## Key Id

The key id is a random value giving identity to the key file. When attaching to a remote drive
for the first time, the key id is associated with the drive to prevent accidental mixing of key files.

## Cipher Key

The cipher key is the key used to perform AES encryption on pack files, it is the key
that ensures data packs cannot be read on the server side.

## HMAC Key

The HMAC key is used to sign the root hashes of the file system datastructures (such as bpy_htree(7)). This
portion of the key ensures data has not been tampered, as the client will refuse any roots with invalid HMAC
signatures.

Here is an example key file contents.
```
{"CipherKey":[82,251,61,142,247,214,36,83,81,180,29,146,11,121,12,58,184,224,143,86,181,253,172,16,15,134,60,48,216,182,122,14],"HmacKey":[55,245,100,160,32,251,132,44,81,162,83,101,98,83,126,138,151,5,15,74,134,139,182,36,1,217,119,238,194,162,104,108],"Id":[250,235,244,145,178,240,15,211,70,146,146,252,162,139,50,70,145,146,162,218,109,110,29,110,50,16,227,221,120,26,130,202]}
```

# See Also

**bpy(1)**, **bpy_ebpack(5)**
