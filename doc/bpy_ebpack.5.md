% bpy_bpack(5)
% Andrew Chambers
% 2016

# NAME

ebpack - ebpack files are encrypted bpy_bpack(5) files

# SYNOPSIS

ebpack files are aes256 encrypted bpack files encryted using a CTR mode cipher. 
The format allows random access decryption with a 32 byte granularity.

The file format consists of a 32 byte random nonce, with N 32 byte blocks of data.
Each block of data is created via the operation ```XOR(AES256ENCRYPT(SECRETKEY, ADD(NONCE, N)), PLAINTEXT)```
and each block of data is accessed via ```XOR(AES256DECRYPT(SECRETKEY, ADD(NONCE, N)), CIPHERTEXT)```.

Because the input data may not be a multiple of 32 bytes, there is always a padded final block.
There maybe be up to 32 bytes of padding in the tail of the final block, starting from the end padding
bytes are 0x00, with a 0x80 byte denoting the final padding byte.

An example of an encrypted file on disk is shown in the following diagram, and requires decryption before
it can be accessed.

```
+-------------+
| Nonce[32]   |
+-------------+
| Block1[32]  |
+-------------+
| Block2[32]  |
+-------------+
| Block3[32]  |
+-------------+
.             .
.    ....     .
.             .
+-------------+
| BlockN[32]  |
+-------------+

```

# SEE ALSO

**bpy(1)** **bpy_bpack(5)**
