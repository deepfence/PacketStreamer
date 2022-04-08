[![Book](https://img.shields.io/website?url=https%3A%2F%2Fdeepfence.github.io%2FPacketStreamer%2F)](https://deepfence.github.io/PacketStreamer/)
[![GitHub license](https://img.shields.io/github/license/deepfence/PacketStreamer)](https://github.com/deepfence/PacketStreamer/blob/master/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/deepfence/PacketStreamer)](https://github.com/deepfence/PacketStreamer/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/deepfence/PacketStreamer)](https://github.com/deepfence/PacketStreamer/issues)
[![Slack](https://img.shields.io/badge/slack-@deepfence-blue.svg?logo=slack)](https://join.slack.com/t/deepfence-community/shared_invite/zt-podmzle9-5X~qYx8wMaLt9bGWwkSdgQ)

# PacketStreamer

Deepfence PacketStreamer is a high-performance remote packet capture and
collection tool. It is used by Deepfence's [ThreatStryker](https://deepfence.io/threatstryker/)
security observability platform to gather network traffic on demand from cloud
workloads for forensic analysis.

Primary design goals:

* Stay light, capture and stream, no additional processing
* Portability, works across virtual machines, Kubernetes and AWS Fargate. Linux
  and Windows.

PacketStreamer **sensors** are started on the target servers. Sensors capture
traffic, apply filters, and then stream the traffic to a central reciever.
Traffic streams may be compressed and/or encrypted using TLS.

The PacketStreamer **receiver** accepts PacketStreamer streams from multiple
remote sensors, and writes the packets to a local `pcap` capture file

<p align="center"><img src="https://raw.githubusercontent.com/deepfence/PacketStreamer/main/images/readme/packetstreamer.png"/><p>

PacketStreamer sensors collect raw network packets on remote hosts. It selects packets
to capture using a BPF filter, and forwards them to a central reciever process
where they are written in pcap format.  Sensors are very lightweight and impose
little performance impact on the remote hosts. PacketStreamer sensors can be
run on bare-metal servers, on Docker hosts, and on Kubernetes nodes.

The PacketStreamer receiver accepts network traffic from multiple sensors,
collecting it into a single, central `pcap` file.  You can then process the 
pcap file or live feed the traffic to the tooling of your choice, such as
`Zeek`, `Wireshark` `Suricata`, or as a live stream for Machine Learning models.

## When to use PacketStreamer

PacketStreamer meets more general use cases than existing alternatives. For
example, [PacketBeat](https://github.com/elastic/beats/tree/master/packetbeat)
captures and parses the packets on multiple remote hosts, assembles
transactions, and ships the processed data to a central ElasticSearch
collector. [ksniff](https://github.com/eldadru/ksniff) captures raw packet
data from a single Kubernetes pod.

Use PacketStreamer if you need a lightweight, efficient method to collect raw
network data from multiple machines for central logging and analysis.

## Who uses PacketStreamer?

 * Deepfence [ThreatStryker](https://deepfence.io/threatstryker/) uses
   PacketStreamer to capture traffic from production platforms for forensics
   and anomaly detection.

## Get in touch

Thank you for using PacketStreamer.

* [<img src="https://img.shields.io/badge/slack-@deepfence-brightgreen.svg?logo=slack">](https://join.slack.com/t/deepfence-community/shared_invite/zt-podmzle9-5X~qYx8wMaLt9bGWwkSdgQ) Got a question, need some help?  Find the Deepfence team on Slack
* https://github.com/deepfence/PacketStreamer/issues: Got a feature request or found a bug? Raise an issue
* [productsecurity *at* deepfence *dot* io](SECURITY.md): Found a security issue? Share it in confidence
* Find out more at [deepfence.io](https://deepfence.io/)

## Security and Support

For any security-related issues in the PacketStreamer project, contact [productsecurity *at* deepfence *dot* io](SECURITY.md).

Please file GitHub issues as needed, and join the Deepfence Community [Slack channel](https://join.slack.com/t/deepfence-community/shared_invite/zt-podmzle9-5X~qYx8wMaLt9bGWwkSdgQ).

## License

The Deepfence PacketStreamer project (this repository) is offered under the [Apache2 license](https://www.apache.org/licenses/LICENSE-2.0).

[Contributions](CONTRIBUTING.md) to Deepfence PacketStreamer project are similarly accepted under the Apache2 license, as per [GitHub's inbound=outbound policy](https://docs.github.com/en/github/site-policy/github-terms-of-service#6-contributions-under-repository-license).
