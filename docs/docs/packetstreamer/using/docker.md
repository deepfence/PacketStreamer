---
title: Run on Docker
---

# Using with Docker

## Build a Container Image

Use the `docker-image` target to build a container image:

```bash
make docker-image

# Alternatively, build a stripped release binary
make docker-image RELEASE=1
```

## Testing on Docker

PacketStreamer container images can be tested locally on Docker.

### Receiver side

```bash
docker run --rm -it \
    -v $(pwd)/contrib/config:/etc/packetstreamer \
    -v $HOME/container_tmp:/tmp \
    -p 8081:8081 \
    deepfenceio/deepfence_packetstreamer \
    receiver --config /etc/packetstreamer/receiver.yaml
```

### Sensor side

```bash
docker run --rm -it \
    --cap-add=NET_ADMIN --net=host \
    -v $(pwd)/contrib/config:/etc/packetstreamer \
    deepfenceio/deepfence_packetstreamer \
    sensor --config /etc/packetstreamer/sensor-local.yaml
```

The sensor requires `--net=host` and `NET_ADMIN` capability in order to capture
all of the packets on the host.

```bash
echo 'GET http://some_ip:80' | vegeta attack -rate 100 -duration 5m | tee results.bin | vegeta report
```

The `pcap` dump file is available in `$HOME/container_tmp/dump_file`.
