# Configuration

`packetstreamer` is configured using a yaml-formatted configuration file.

```yaml
input:                         # required in 'receiver' mode
  address: _ip-address_
  port: _listen-port_
output:
  server:                      # required in 'sensor' mode
    address: _ip-address_
    port: _listen-port_
  file:                        # for 'receiver' mode
    path: _filename_|stdout    # 'stdout' is a reserved name. Receiver will write to stdout
  s3:                          # for 'receiver' mode
    bucket: _name_
    region: _region_
    keyFormat: _date_format_   # default: /packetstreamer/%Y/%m/%d/%H/%M/%S
    totalFileSize: _size_      # example: 5M, 500K; default: 1M
    uploadTimeout: _time_      # example: 30s, 2m; default: 1m
tls:                           # optional
  enable: _true_|_false_
  certfile: _filename_
  keyfile: _filename_
auth:                          # optional; receiver and sensor must use same shared key
  enable: _true_|_false_
  key: _string_
compressBlockSize: _integer_   # optional; default: 65
inputPacketLen: _integer_      # optional; default: 65535
logFilename: _filename_        # optional
pcapMode: _Allow_|_Deny_|_All_ # optional
capturePorts: _list-of-ports_  # optional
captureInterfacesPorts: _map: interface-name:port_ # optional
ignorePorts: _list-of-ports_   # optional
```

You can find example configuration files in the [`/contrib/config/`](https://github.com/deepfence/PacketStreamer/tree/main/contrib/config)
folder.
