# Using on Vagrant

On a single host, you may use [Vagrant](https://www.vagrantup.com) to run
sensor and receiver hosts easily:

Install Vagrant according to [the official instructions](https://www.vagrantup.com/downloads).
By default, Vagrant uses Virtualbox; you should install
[vagrant-libvirt](https://github.com/vagrant-libvirt/vagrant-libvirt), using
`vagrant plugin install vagrant-libvirt`.

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
