---
title: Configuration
---

# Configuration

`packetstreamer` is configured using a yaml-formatted configuration file.

```yaml
input:                             # required in 'receiver' mode
  address: ip-address
  port: listen-port
output:
  server:                          # required in 'sensor' mode
    address: ip-address
    port: listen-port
  file:                            # required in 'receiver' mode
    path: filename|stdout          # 'stdout' is a reserved name. Receiver will write to stdout
  plugins:                         # optional
    s3:
      bucket: string
      region: string
      totalFileSize: filesize      # optional; default: 10 MB
      uploadChunkSize: filesize    # optional; default: 5 MB
      uploadTimeout: timeout       # optional; default: 1m
      cannedACL: acl               # optional; default: Bucket owner enforced
tls:                               # optional
  enable: true|false
  certfile: filename
  keyfile: filename
auth:                              # optional; receiver and sensor must use same shared key
  enable: true|false
  key: string
compressBlockSize: integer         # optional; default: 65
inputPacketLen: integer            # optional; default: 65535
gatherMaxWaitSec: integer          # optional; default: 5
logFilename: filename              # optional
pcapMode: Allow|Deny|All           # optional
capturePorts: list-of-ports        # optional
captureInterfacesPorts: map: interface-name:port # optional
ignorePorts: list-of-ports         # optional
```

You can find example configuration files in the [`/contrib/config/`](https://github.com/deepfence/PacketStreamer/tree/main/contrib/config)
folder.
