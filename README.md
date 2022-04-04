[![GitHub license](https://img.shields.io/github/license/deepfence/PacketStreamer)](https://github.com/deepfence/PacketStreamer/blob/master/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/deepfence/PacketStreamer)](https://github.com/deepfence/PacketStreamer/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/deepfence/PacketStreamer)](https://github.com/deepfence/PacketStreamer/issues)
[![Slack](https://img.shields.io/badge/slack-@deepfence-blue.svg?logo=slack)](https://join.slack.com/t/deepfence-community/shared_invite/zt-podmzle9-5X~qYx8wMaLt9bGWwkSdgQ)


# PacketStreamer

Deepfence PacketStreamer is a high-performance remote packet capture and collection tool. It is used by Deepfence's [ThreatStryker](https://deepfence.io/threatstryker/) security observability platform to gather network traffic on demand from cloud workloads for forensic analysis.

Primary design goals:

* Stay light, capture and stream, no additional processing
* Portability, works across virtual machines, Kubernetes and AWS Fargate. Linux and Windows.

PacketStreamer **sensors** are started on the target servers. Sensors capture traffic, apply filters, and then stream the traffic to a central reciever. Traffic streams may be compressed and/or encrypted using TLS.

The PacketStreamer **reciever** accepts PacketStreamer streams from multiple remote sensors, and writes the packets to a local `pcap` capture file

<p align="center"><img src="/images/readme/packetstreamer.png" width="66%"/><p>

PacketStreamer sensors collect raw network packets on remote hosts.  It selects packets to capture using a BPF filter, and forwards them to a central reciever process where they are written in pcap format.  Sensors are very lightweight and impose little performance impact on the remote hosts.  PacketStreamer sensors can be run on bare-metal servers, on Docker hosts, and on Kubernetes nodes.

The PacketStreamer receiver accepts network traffic from multiple sensors, collecting it into a single, central `pcap` file.  You can then process the pcap file or live feed the traffic to the tooling of your choice, such as `Zeek`, `Wireshark` `Suricata`, or as a live stream for Machine Learning models.

### When to use PacketStreamer

PacketStreamer meets more general use cases than existing alternatives. For example, [PacketBeat](https://github.com/elastic/beats/tree/master/packetbeat) captures and parses the packets on multiple remote hosts, assembles transactions, and ships the processed data to a central ElasticSearch collector. [ksniff](https://github.com/eldadru/ksniff) captures raw packet data from a single Kubernetes pod.

Use PacketStreamer if you need a lightweight, efficient method to collect raw network data from multiple machines for central logging and analysis.

### Who uses PacketStreamer?

 * Deepfence [ThreatStryker](https://deepfence.io/threatstryker/) uses PacketStreamer to capture traffic from production platforms for forensics and  anomaly detection.


# Quickstart: Build and Run PacketStreamer

## Building PacketStreamer

Build the `packetstreamer` binary using the `go` toolchain as follows:

```bash
make
```

## Run a PacketStreamer receiver

```bash
packetstreamer receiver --config [configuration_file]
```

You can use an example configuration file:

```bash
packetstreamer receiver --config ./contrib/config/receiver-local.yaml
```

You can process the `pcap` output in a variety of ways:

```bash
# pass the output file /tmp/dump_file to tcpdump:
tail -c +1 -f /tmp/dump_file | tcpdump -r -
```

```bash
# Edit the configuration to write to /dev/stdout, and pipe output to tcpdump:
./packet-streamer receiver --config contrib/receiver-stdout.yaml | tcpdump -r -
```

## Run a PacketStreamer sensor

```bash
sudo packetstreamer sensor --config [configuration_file]
```

You can use an example configuration file:

```bash
sudo packetstreamer sensor --config ./contrib/config/sensor-local.yaml
```

When running the sensor remotely, edit the configuration file to target the remote receiver.

## Testing PacketStreamer

You can generate some test traffic using your favorite load generator - `ab`, `wrk`, `httperf`, `vegeta`.  For example, to use vegeta:

```bash
# install vegeta
go install github.com/tsenart/vegeta@latest

echo 'GET http://some_ip:80' | vegeta attack -rate 100 -duration 5m | tee results.bin | vegeta report
```

# Advanced Build Options

Use the `RELEASE` parameter to strip the binary for a production environment:

```bash
make RELEASE=1
```

Use the `STATIC` parameter to statically-link the binary:

```bash
make STATIC=1
```

## Build using Docker

Use the `docker-bin` target to build `packetstreamer` with Docker. The binary will be statically linked with `musl` and `libpcap`, making it portable across Linux distributions:

```bash
make docker-bin

# Alternatively, build a stripped release binary
make docker-bin RELEASE=1
```

## Build a Container Image

Use the `docker-image` target to build a container image:

```bash
make docker-image

# Alternatively, build a stripped release binary
make docker-image RELEASE=1
```

## Deploy on Kubernetes

PacketStreamer can be deployed on Kubernetes using Helm:

```bash
kubectl apply -f ./contrib/kubernetes/namespace.yaml
helm install packetstreamer ./contrib/helm/ --namespace packetstreamer
```

By default, the Helm chart deploys a DaemonSet with sensor on all nodes and one receiver instance. For the custom configuration values, please refer to the [values.yaml file](/contrib/helm/values.yaml).

# Advanced Test Scenarios

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

The sensor requires `--net=host` and `NET_ADMIN` capability in order to capture all of the packets on the host.

```bash
echo 'GET http://some_ip:80' | vegeta attack -rate 100 -duration 5m | tee results.bin | vegeta report
```

The `pcap` dump file is available in `$HOME/container_tmp/dump_file`.

## Testing on Vagrant

On a single host, you may use [Vagrant](https://www.vagrantup.com) to run sensor and receiver hosts easily:

Install Vagrant according to [the official instructions](https://www.vagrantup.com/downloads). By default, Vagrant uses Virtualbox; you should install [vagrant-libvirt](https://github.com/vagrant-libvirt/vagrant-libvirt), using `vagrant plugin install vagrant-libvirt`.

Start the two Vagrant VMs, `receiver` and `sensor`:

```bash
vagrant up

vagrant status
# Current machine states:
#
# receiver                  running (libvirt)
# sensor                    running (libvirt)
```

SSH to those VMs (in separate terminals) by using the following commands:

```bash
vagrant ssh receiver
```

```bash
vagrant ssh sensor
```

On each, enter the source code directory:

### Receiver side

```bash
cd PacketStreamer
./packetstreamer receiver --config ./contrib/config/receiver-vagrant.yaml
```

### Sensor side

```bash
cd PacketStreamer
sudo ./packetstreamer --config ./contrib/config/sensor-vagrant.yaml
```

Generate some live traffic

```bash
echo 'GET http://some_ip:80' | vegeta attack -rate 100 -duration 5m | tee results.bin | vegeta report
```

# PacketStreamer Configuration

`packetstreamer` is configured using a yaml-formatted configuration file.

```yaml
input:                         # required in 'receiver' mode
  address: _ip-address_
  port: _listen-port_
output:
  server:                      # required in 'sensor' mode
    address: _ip-address_
    port: _listen-port_
  file:                        # required in 'receiver' mode
    path: _filename_
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

You can find example configuration files in the [`/contrib/config/`](contrib/config/) folder.

# Get in touch

Thank you for using PacketStreamer.

* [<img src="https://img.shields.io/badge/slack-@deepfence-brightgreen.svg?logo=slack">](https://join.slack.com/t/deepfence-community/shared_invite/zt-podmzle9-5X~qYx8wMaLt9bGWwkSdgQ) Got a question, need some help?  Find the Deepfence team on Slack
* https://github.com/deepfence/PacketStreamer/issues: Got a feature request or found a bug?  Raise an issue
* [productsecurity at deepfence dot io](SECURITY.md): Found a security issue?  Share it in confidence
* Find out more at [deepfence.io](https://deepfence.io/)

# Security and Support

For any security-related issues in the PacketStreamer project, contact [productsecurity *at* deepfence *dot* io](SECURITY.md).

Please file GitHub issues as needed, and join the Deepfence Community [Slack channel](https://join.slack.com/t/deepfence-community/shared_invite/zt-podmzle9-5X~qYx8wMaLt9bGWwkSdgQ).

# License

The Deepfence PacketStreamer project (this repository) is offered under the [Apache2 license](https://www.apache.org/licenses/LICENSE-2.0).

[Contributions](CONTRIBUTING.md) to Deepfence PacketStreamer project are similarly accepted under the Apache2 license, as per [GitHub's inbound=outbound policy](https://docs.github.com/en/github/site-policy/github-terms-of-service#6-contributions-under-repository-license).
