# Using locally

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
# Edit the configuration to write to the special name 'stdout', and pipe output to tcpdump:
./packet-streamer receiver --config ./contrib/config/receiver-stdout.yaml | tcpdump -r -
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
