# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "generic/ubuntu2110"
  config.vm.synced_folder ".", "/home/vagrant/PacketStreamer"

  config.vm.provider "libvirt" do |libvirt, override|
    override.vm.synced_folder ".", "/home/vagrant/PacketStreamer", type: "nfs", mount_options: ["vers=3,tcp"]
  end

  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y \
      build-essential \
      golang \
      libpcap-dev
    systemctl stop ufw
    systemctl disable ufw
  SHELL

  config.vm.provision "shell", privileged: false, inline: <<-SHELL
    echo 'export PATH="/home/vagrant/go/bin:$PATH"' >> /home/vagrant/.bashrc
    go install github.com/tsenart/vegeta@latest
  SHELL

  config.vm.define "receiver" do |r|
    r.vm.network "private_network", ip: "192.168.33.10"
  end

  config.vm.define "sensor" do |s|
    s.vm.network "private_network", ip: "192.168.33.11"
  end
end
