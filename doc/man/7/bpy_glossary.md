% bpy_glossary(7)
% Andrew Chambers
% 2016

# Name

bpy glossary

# Synopsis

This page is a glossary of terms that might be useful for understanding the buppy.io tools.

## Bpack

A file used to store a set of arbitrary keys and values used by bpy to construct a virtual file system. Fully described
at bpy_bpack(5).

## Chunk

A chunk refers to an arbitrary block of binary stored within a bpy_bpack(5) file.

## Drive

A drive is a collection of encrypted packfiles containing a bpy_fs(3). It also has
other data associated with it, including a place to store the hash of root ref.

## Ebpack

An encrypted bpack file, described at bpy_epbpack(5)

## Hash

In buppy context, hash refers to a sha256 sum. Buppy internally works as https://en.wikipedia.org/wiki/Content-addressable_storage, 
the hash of a chunk of data represents its 

## HTree

HTree is short for hash tree, and is a format for storing a stream of data in limited sized chunks, it is fully described
at bpy_htree(5).

## Ref

A ref is a data structure stored inside the bpy content store, forming a singly linked list of a drive's history.
It is described in more detail at bpy_ref(5).

## Root

A root refers to either the root of an htree, or to the root ref stored as the entry point to a bpy drive.

# See Also

**bpy(1)**
