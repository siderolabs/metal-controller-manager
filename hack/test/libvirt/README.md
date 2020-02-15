# Integration Testing

## Setup

### Prerequisites

- A linux machine with KVM enabled
- `docker`
- `docker-compose`
- `virt-install`
- `qemu-kvm`

```bash
apt install -y virtinst qemu-kvm
```

### Start iPXE, and Dnsmasq

```bash
docker-compose up
```

> Note: This will run all services in the foreground.

### Create the Management Plane

```bash
osctl cluster create --name arges --masters=1 --workers=0 --cidr 172.28.0.0/16 --image docker.io/autonomy/talos:v0.3.2 --wait
osctl kubeconfig -f ./ipxe/assets/arges
kubectl --kubeconfig ./ipxe/assets/arges/kubeconfig apply -f ../../config/crd/metal.talos.dev_machineinventories.yaml
```

### Create the VMs

```bash
./libvirt.sh up
```

### Getting the Console Logs

```bash
virsh console integration-test
```

### Connecting to the Nodes

#### From the Host

##### Setup DNS

Append the following to `/etc/hosts`:

```text
172.28.1.3 kubernetes.talos.dev
172.28.1.10 control-plane-1.talos.dev
172.28.1.11 control-plane-2.talos.dev
172.28.1.12 control-plane-3.talos.dev
172.28.1.13 worker-1.talos.dev
```

##### Setup `osctl` and `kubectl`

```bash
export TALOSCONFIG=$PWD/matchbox/assets/talosconfig
export KUBECONFIG=$PWD/matchbox/assets/kubeconfig
```

```bash
osctl config endpoint 172.28.1.10
osctl kubeconfig ./matchbox/assets/kubeconfig
```

#### From a Container

```bash
./libvirt.sh workspace
```

```bash
osctl config endpoint 172.28.1.10
osctl kubeconfig .
```

#### Verify Connectivity

```bash
osctl services
kubectl get nodes
```

## Teardown

To teardown the test:

```bash
docker-compose down
./libvirt.sh down
```
