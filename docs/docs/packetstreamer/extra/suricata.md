---
title: Analyse with Suricata
---

# Suricata

PacketStreamer's output can be used for threat detection with Suricata.

## From file

If receiver is configured with the `File` output, the file can be used as an
input to Suricata with the following commannd.

```bash
tail -f /tmp/dump_file | suricata -v -c /etc/suricata/suricata.yaml -r /dev/stdin
```

It assumes that:

* Suricata's configuration file is `/etc/suricata/suricata.yaml`.
* PacketStreamer receiver is configured with output file to `/tmp/dump_file`.

Example receiver configuration:

```yaml
input:
  address: 0.0.0.0
  port: 8081
output:
  file:
    path: /tmp/dump_file
```

## From stdout

When PacketStreamer writes to stdout, the output can be directly piped to
Suricata:

```bash
./packet-streamer receiver --config ./contrib/config/receiver-stdout.yaml | suricata -v -c /etc/suricata/suricata.yaml -r /dev/stdin
```
