% bpy_trust(7)
% Andrew Chambers
% 2017

# Synopsis

This document is a brief summary of the trust model that buppy.io uses for
keeping your data safe. It describes what the bpy tool does and does not do to protect 
your data and privacy.

# The command line client

The command line client is the tool installed on the users computer. The user
can trust this tool to act on their behalf because all the source code
is avaliable for inspection and modification. Official releases are signed with
signify and can be verified by the user, but the user can still compile their own
version from source and use that.

Therefore the client can be trusted to:

- Encrypt all file data in the bpy_ebpack(5) format with the users local key before it is sent to the server.
- Encrypt all messages sent between the client and server using ssh-2 as a transport protocol.
- Encrypt and sign all drive roots with a key before sending them to the server.
- Verify all signatures of data retrieved before attempting to decrypt it, and alerting the user of any signature failures.
- Store files on the local system such that only the current user can access them.
- Cache as data locally where possible to limit weaknesses from data access pattern analysis from the server side.

The client *DOES NOT*:

- Encrypt cached user data on the local machine. It is assumed the users local machine is trusted. For additional security consider
  using whatever disk encryption your operating system provides.
- Encrypt your key file on your disk, it is assumed the users machine is trusted by the the user.

# The buppy.io server

The design is such that if the buppy.io servers are compromised, the attacker will only gain access to encrypted pack files, and public keys only.
The aim being that at worst an attacker can cause data loss by file deletion - but no data disclosure or tampering should be possible.

The user is trusting the server to:

- Ensure drive roots have incrementing 'version' fields to prevent replay attacks.
- Store encrypted pack files for later retrieval, though still verifying the contents.
- Encrypt all messages sent between the client and server using ssh-2 as a transport protocol.
- Store ssh public keys, and block access to public keys that are not authorized to access a drive.



# See Also

**bpy(1)**
