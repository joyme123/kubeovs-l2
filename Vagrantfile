# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
    config.vm.box = "jiang/centos7_ovs"
    config.vm.box_check_update = false
    config.vm.network "public_network", ip: "192.168.50.199"
    config.vm.synced_folder ".", "/home/vagrant/app"
    config.vm.provider "virtualbox" do |vb|
       vb.memory = "2048"
       vb.customize ["modifyvm", :id, "--nicpromisc2", "allow-all"]
       vb.customize ["modifyvm", :id, "--nicpromisc3", "allow-all"]
    end
end
