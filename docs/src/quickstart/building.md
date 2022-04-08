# Building

Build the `packetstreamer` binary using the `go` toolchain as follows:

```bash
make
```

## Advanced Build Options

Use the `RELEASE` parameter to strip the binary for a production environment:

```bash
make RELEASE=1
```

Use the `STATIC` parameter to statically-link the binary:

```bash
make STATIC=1
```

## Build using Docker

Use the `docker-bin` target to build `packetstreamer` with Docker. The binary
will be statically linked with `musl` and `libpcap`, making it portable across
Linux distributions:

```bash
make docker-bin

# Alternatively, build a stripped release binary
make docker-bin RELEASE=1
```
