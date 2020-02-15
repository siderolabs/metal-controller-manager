#!/bin/bash

set -e

if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

function main {
  case "$1" in
    "up") up;;
    "down") down;;
    *)
      usage
      exit 2
      ;;
  esac
}

function usage {
  echo "USAGE: ${0##*/} <command>"
  echo "Commands:"
  echo -e "\up\t\spin up QEMU/KVM nodes on the arges0 bridge"
  echo -e "\down\t\tear down the QEMU/KVM nodes"
  echo -e "\workspace\t\run and enter a docker container ready for osctl and kubectl use"
}

NODES=(integration-test)

VM_MEMORY=${VM_MEMORY:-512}
VM_DISK=${VM_DISK:-1}

COMMON_VIRT_OPTS="--memory=${VM_MEMORY} --cpu=host --vcpus=1 --disk pool=default,size=${VM_DISK} --os-type=linux --os-variant=generic --noautoconsole --graphics none --events on_poweroff=preserve --rng /dev/urandom"

CONTROL_PLANE_1_NAME=integration-test
CONTROL_PLANE_1_MAC=52:54:00:a1:9c:ae

function up {
    virt-install --name $CONTROL_PLANE_1_NAME --network=bridge:arges0,model=e1000,mac=$CONTROL_PLANE_1_MAC $COMMON_VIRT_OPTS --boot=hd,network
}

function down {
    for node in ${NODES[@]}; do
      virsh destroy $node
    done
    for node in ${NODES[@]}; do
      virsh undefine $node
    done
    virsh pool-refresh default
    for node in ${NODES[@]}; do
      virsh vol-delete --pool default $node.qcow2
    done
}

main $@
