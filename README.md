# pf9ctl

### Status
![Go](https://github.com/platform9/pf9ctl/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/platform9/pf9ctl)](https://goreportcard.com/report/github.com/platform9/pf9ctl)

### Purpose
   CLI tool for Platform9 management.
   
### Requirements(Prerequisites)

- CPUs: Minimum 2 CPUs needed on host
- RAM: 12 GB 
- Disk: At least 30 GB of total disk space and 15 GB of free space is needed on host
- Sudo access to the user
- OS(Supported) : 
    - Ubuntu (16.04,18.04,20.04)
    - RHEL/Centos (7.x)

### Proxy support

The CLI allows configuration where all HTTPS requests can be routed through a proxy. See the `Configuration` section to see how to configure the proxy URL.

### Non-interative mode

The CLI can be run in a non-interactive mode with flag `--no-prompt`. Using this disables all user prompts. If required flags are not passed to a sub-command or in case of any error, the CLI returns with a non zero code.

### Usage
- Downloading the CLI 
```sh
bash <(curl -sL https://pmkft-assets.s3-us-west-1.amazonaws.com/pf9ctl_setup) 
```
- **Help** 
```sh
#pf9ctl --help

CLI tool for Platform9 management.
	Platform9 Managed Kubernetes cluster operations. Read more at
	http://pf9.io/cli_clhelp.

Usage:
  pf9ctl [command]

Available Commands:
  attach-node           Attaches a node to the Kubernetes cluster
  authorize-node        Authorizes this node with PMK control plane
  bootstrap             Creates a single-node Kubernetes cluster using the current node
  bundle                Gathers the support bundle and uploads it to S3
  check-amazon-provider Checks if the user has Amazon cloud permissions
  check-azure-provider  Checks if the user has Azure cloud permissions
  check-google-provider Checks if the user has Google cloud permissions
  check-node            Checks prerequisites on a node to use with PMK
  config                Creates or get the config
  deauthorize-node      Deauthorizes this node from the PMK control plane
  decommission-node     Decommissions this node from the PMK control plane
  delete-cluster        Deletes the cluster
  detach-node           Detaches a node from a Kubernetes cluster
  help                  Help about any command
  prep-node             Sets up prerequisites & prepares a node to use with PMK
  upgrade               Checks for a new version of the CLI
  version               Prints current version of CLI being used

Flags:
  -h, --help             help for pf9ctl
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs

Use "pf9ctl [command] --help" for more information about a command.
```
- **Version**

  **This command is used to get the current version of the CLI**
```sh
#pf9ctl version

pf9ctl version: v1.16

```
- **Upgrading**

  **This command is used upgrade the CLI to its newest version if there is one**
```sh
#pf9ctl upgrade
You already have the newest version
```   
```sh
#pf9ctl upgrade
New version found. Please upgrade to the newest version
Do you want to upgrade? (y/n): y

Downloading the CLI

Installing the CLI
Successfully updated.
```

```sh
#pf9ctl upgrade --skip-check
New version found. Please upgrade to the newest version

Downloading the CLI

Installing the CLI
Successfully updated.
```

- **Configuration**

  This is used to setup or get the control-plane configuration. It includes the DU FQDN , username, region and the tenant(service).

```sh
#pf9ctl config

Create or get PF9 controller config used by this CLI

Usage:
  pf9ctl config [command]

Available Commands:
  get         Print stored config
  set         Create a new config

Flags:
  -h, --help   help for config

Global Flags:
      --no-prompt   disable all user prompts
      --verbose   print verbose logs

Use "pf9ctl config [command] --help" for more information about a command.
```  


```sh
#pf9ctl config set --help

We can set config through prompt or with flags. 

Create a new config that can be used to query Platform9 controller

Usage:
  pf9ctl config set [flags]

Flags:
  -u, --account-url string   sets account_url
  -h, --help                 help for set
      --mfa string           set MFA token
  -p, --password string      sets password (use 'single quotes' to pass password)
  -l, --proxy-url string     sets proxy URL, can be specified as [<protocol>][<username>:<password>@]<host>:<port>
  -r, --region string        sets region
  -t, --tenant string        sets tenant
  -e, --username string      sets username

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs
```  


- **Check Node**

  This command will perform the prerequisite check before a node can be added to the cluster. It checks for the following:
  
     - Mandatory Checks
        - Existing pf9 packages
        - Installing missing packages
        - Sudo Access Check
        - Ports Check
        - OS Check
        - Existing Kubernetes Cluster Check
        - Check execute permissions on /tmp folder
        - Disabling swap and removing swap in fstab
  
     - Optional Checks
       - Removal of existing pf9ctl(Python Based CLI)
       - Resources(CPU,Disk,Memory check)
```sh
#pf9ctl check-node --help
Check if a node satisfies prerequisites to be ready to be added to a Kubernetes cluster. Read more
	at https://platform9.com/blog/support/managed-container-cloud-requirements-checklist/

Usage:
  pf9ctl check-node [flags]

Flags:
  -h, --help               help for check-node
  -i, --ip strings         IP address of host to be prepared
      --mfa string         MFA token
  -p, --password string    ssh password for the nodes (use 'single quotes' to pass password)
  -s, --ssh-key string     ssh key file for connecting to the nodes
  -e, --sudo-pass string   sudo password for user on remote host
  -u, --user string        ssh username for the nodes

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs
```

   **Check-Node(Local)**
   
 ```sh
#pf9ctl check-node

✓ Loaded Config Successfully
✓ Removal of existing CLI
✓ Existing Platform9 Packages Check
✓ Required OS Packages Check
✓ SudoCheck
✓ CPUCheck
✓ DiskCheck
✓ MemoryCheck
✓ PortCheck
✓ Existing Kubernetes Cluster Check

✓ Completed Pre-Requisite Checks successfully
```

   **Check-Node(Remote)**
   
 ```sh
#pf9ctl check-node  -i 10.128.241.203 -u centos

✓ Loaded Config Successfully
You can choose either password or sshKey
Enter 1 for password and 2 for sshKey
Enter Option : 1
Enter password for remote host: 
✓ Removal of existing CLI
✓ Existing Platform9 Packages Check
✓ Required OS Packages Check
✓ SudoCheck
✓ CPUCheck
✓ DiskCheck
✓ MemoryCheck
✓ PortCheck
✓ Existing Kubernetes Cluster Check

✓ Completed Pre-Requisite Checks successfully
```
- **prep-node**

 This command onboards a node. It installs platform9 packages on the host. After completion of this command, the node is available to be managed on the Platform9 control plane.
 ```sh
#pf9ctl prep-node --help
Prepare a node to be ready to be added to a Kubernetes cluster. Read more
	at http://pf9.io/cli_clprep.

Usage:
  pf9ctl prep-node [flags]

Flags:
  -h, --help               help for prep-node
  -i, --ip strings         IP address of host to be prepared
      --mfa string         MFA token
  -p, --password string    ssh password for the nodes (use 'single quotes' to pass password)
  -c, --skip-checks         Will skip optional checks if true
  -s, --ssh-key string     ssh key file for connecting to the nodes
  -e, --sudo-pass string   sudo password for user on remote host
  -u, --user string        ssh username for the nodes

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs

```
  **prep-Node(Local)**
 ```sh
#pf9ctl prep-node

✓ Loaded Config Successfully
✓ Removal of existing CLI
✓ Existing Platform9 Packages Check
✓ Required OS Packages Check
✓ SudoCheck
✓ CPUCheck
✓ DiskCheck
✓ MemoryCheck
✓ PortCheck
✓ Existing Kubernetes Cluster Check

✓ Completed Pre-Requisite Checks successfully

✓ Disabled swap and removed swap in fstab
✓ Platform9 packages installed successfully
✓ Initialised host successfully
✓ Host successfully attached to the Platform9 control-plane
```
  **prep-Node(Remote)**
```sh
#pf9ctl prep-node -i 10.128.241.203 -u centos

✓ Loaded Config Successfully
You can choose either password or sshKey
Enter 1 for password and 2 for sshKey
Enter Option : 1
Enter password for remote host: 
✓ Removal of existing CLI
✓ Existing Platform9 Packages Check
✓ Required OS Packages Check
✓ SudoCheck
✓ CPUCheck
✓ DiskCheck
✓ MemoryCheck
✓ PortCheck
✓ Existing Kubernetes Cluster Check

✓ Completed Pre-Requisite Checks successfully

✓ Disabled swap and removed swap in fstab
✓ Platform9 packages installed successfully
✓ Initialised host successfully
✓ Host successfully attached to the Platform9 control-plane
```
- **bundle(SupportBundle)**

This is used to gather and upload a support bundle(a folder containing logs for pf9 services and pf9ctl) to the S3 location.

```sh
#pf9ctl bundle --help

Gathers support bundle that includes logs for pf9 services and pf9ctl, uploads to S3

Usage:
  pf9ctl bundle [flags]

Flags:
  -h, --help               help for bundle
  -i, --ip strings         IP address of host to be prepared
      --mfa string         MFA token
  -p, --password string    ssh password for the nodes (use 'single quotes' to pass password)
  -s, --ssh-key string     ssh key file for connecting to the nodes
  -e, --sudo-pass string   sudo password for user on remote host
  -u, --user string        ssh username for the nodes

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs

```
   **bundle(Local)**
```sh
#pf9ctl bundle 

✓ Loaded Config Successfully
2021-05-17T06:45:03.95Z	INFO	==========Uploading supportBundle to S3 bucket==========
✓ Succesfully uploaded pf9ctl supportBundle to loguploads.platform9.com bucket at https://s3-us-west-2.amazonaws.com/loguploads.platform9.com/pmkft-1614749234-62656.platform9.io/172.20.7.21/ location 
```
   **bundle(Remote)**
```sh
#pf9ctl bundle -i 10.128.241.203 -u centos

✓ Loaded Config Successfully
You can choose either password or sshKey
Enter 1 for password and 2 for sshKey
Enter Option : 1
Enter password for remote host: 
2021-05-17T06:51:41.338Z	INFO	==========Uploading supportBundle to S3 bucket==========
✓ Succesfully uploaded pf9ctl supportBundle to loguploads.platform9.com bucket at https://s3-us-west-2.amazonaws.com/loguploads.platform9.com/pmkft-1614749234-62656.platform9.io/172.20.7.104/ location 
```

  **attach-node**

```sh
#pf9ctl attach-node --help

Attach nodes to existing cluster. At a time, multiple workers but only one master can be attached

Usage:
  pf9ctl attach-node [flags] cluster-name

Flags:
  -h, --help                help for attach-node
  -m, --master-ip strings   master node ip address
      --mfa string          MFA token
  -u, --uuid string         uuid of the cluster to attach the node to
  -w, --worker-ip strings   worker node ip address

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs
```


```sh
#pf9ctl attach-node -m 172.20.7.66 -w 172.20.7.58 test-cluster
✓ Loaded Config Successfully
2021-05-26T11:58:01.9579Z	INFO	Worker node(s) [bf5364cf-e2fd-4500-97fb-0b01be26084f] attached to cluster
2021-05-26T11:58:03.6328Z	INFO	Master node(s) [615c1042-48a3-42e8-8003-ac135d12e6f4] attached to cluster
```

  **detach-node**

```sh
#pf9ctl detach-node --help
Detach nodes from their clusters. If no ips are sent it will detach the node on which the command was run.

Usage:
  pf9ctl detach-node [flags]

Flags:
  -h, --help              help for detach-node
      --mfa string          MFA token
  -n, --node-ip strings   node ip address

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs
```


```sh
#pf9ctl detach-node
✓ Loaded Config Successfully
Starting detaching process
2021-11-08T08:35:14.1182Z	INFO	Node [9cfe32a2-6518-4b63-a55b-f9a9c1148e6a] detached from cluster

```

```sh
#pf9ctl detach-node -n ip
✓ Loaded Config Successfully
Starting detaching process
2021-11-08T08:35:14.1182Z	INFO	Node [9cfe32a2-6518-4b63-a55b-f9a9c1148e6a] detached from cluster

```

```sh
#pf9ctl detach-node -n ip1,ip2
✓ Loaded Config Successfully
Starting detaching process
2021-11-08T08:35:14.1182Z	INFO	Node [9cfe32a2-6518-4b63-a55b-f9a9c1148e6a] detached from cluster
2021-11-08T08:35:14.1182Z	INFO	Node [691a9feb-6b62-4235-bf27-208a14744843	 detached from cluster

```

  **deauthorize-node**

```sh
#pf9ctl deauthorize-node --help
Deauthorizes this node. It will warn the user if the node was a master node or a part of a single node cluster.

Usage:
  pf9ctl deauthorize-node [flags]

Flags:
  -h, --help         help for deauthorize-node
  -i, --ip string    IP address of the host to be deauthorized
      --mfa string   MFA token

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs

```

```sh
#pf9ctl deauthorize-node
✓ Loaded Config Successfully
Node deauthorization started....This may take a few minutes....Check the latest status in UI
```


  **authorize-node**

```sh
#pf9ctl authorize-node --help
Authorizes this node

Usage:
  pf9ctl authorize-node [flags]

Flags:
  -h, --help         help for authorize-node
  -i, --ip string    IP address of the host to be authorized
      --mfa string   MFA token

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs
```

```sh
#pf9ctl authorize-node
✓ Loaded Config Successfully
Node authorization started....This may take a few minutes....Check the latest status in UI
```

  **delete-cluster**

```sh
#pf9ctl delete-cluster --help
Deletes the cluster with the specified name. Additionally the user can pass the cluster UID instead of the name.

Usage:
  pf9ctl delete-cluster [flags]

Flags:
  -h, --help          help for delete-cluster
      --mfa string    MFA token
  -n, --name string   clusters name
  -i, --uuid string   clusters uuid

Global Flags:
      --no-prompt disable all user prompts
      --verbose   print verbose logs

```

```sh
#pf9ctl delete-cluster
2021-11-08T10:25:22.544Z	FATAL	You must pass a cluster name or the cluster uuid
```

```sh
#pf9ctl delete-cluster -n ClusterName
✓ Loaded Config Successfully
Cluster deletion started....This may take a few minutes.
```

```sh
#pf9ctl delete-cluster -i 023be0b0-1348-4d8a-a9b7-25bd4293cbbd
✓ Loaded Config Successfully
Cluster deletion started....This may take a few minutes.
```


  **decommission-node**

```sh
#pf9ctl decommission-node --help
Removes the host agent package and decommissions this node from the Platform9 control plane.

Usage:
  pf9ctl decommission-node [flags]

Flags:
  -h, --help              help for decommission-node
  -i, --ip strings        IP address of host to be decommissioned
      --mfa string        MFA token
  -p, --password string   ssh password for the nodes (use 'single quotes' to pass password)
  -s, --ssh-key string    ssh key file for connecting to the nodes
  -u, --user string       ssh username for the nodes

Global Flags:
      --log-dir string   path to save logs
      --no-prompt        disable all user prompts
      --verbose          print verbose logs
```

When node is connected to a cluster:
```sh
#pf9ctl decommission-node
✓ Loaded Config Successfully
Node is attached to test-2 cluster
2024-05-03T08:58:57.4328Z	FATAL	Node is still attached to a cluster. Please run detach-node command first and wait for the node to be completely removed from the cluster and only then run decommision-node command
```

When node is not connected to any cluster:
```sh
#pf9ctl decommission-node
✓ Loaded Config Successfully
Deauthorized node from UI
Removing pf9-hostagent (this might take a few minutes...)
Removed hostagent
Removing logs...
Running clean all
Removing pf9 HOME dir
Node decommissioning started....This may take a few minutes....Check the latest status in UI
```

  **check-amazon-provider**
```sh
#pf9ctl check-amazon-provider -i iamUser -a access-key -s secret-key -r us-east-1

✓ ELB Access
✓ Route53 Access
✓ Availability Zones success
✓ EC2 Access
✓ VPC Access
✓ IAM Access
✓ Autoscaling Access
✓ EKS Access
```
  **check-google-provider**
```sh
#pf9ctl check-google-provider -p /home/duser/Downloads/service-account.json -n testProject -e user@email.com

✓  Success roles/iam.serviceAccountUser
✓  Failed roles/container.admin
✓  Failed roles/compute.viewer
✓  Success roles/viewer
```

  **check-azure-provider**
```sh
#pf9ctl check-google-provider -t tenantID -c clientID -s subscriptionID -k secretKey

✓ Has access
```

 **bootstrap**

 ```sh
#pf9ctl bootstrap --help
Bootstrap a single node Kubernetes cluster with current node as the master node.

Usage:
  pf9ctl bootstrap [flags] cluster-name

Examples:
pf9ctl bootstrap <clusterName> --pmk-version <version>

Required Flags:
		--pmk-version string                  Kubernetes pmk version
Optional Flags:
		--advanced-api-configuration string   Allowed API groups and version. Option: default, all & custom
		--allow-workloads-on-master           Taint master nodes ( to enable workloads ), use either --allow-workloads-on-master or --allow-workloads-on-master=false to change (default true)
		--api-server-flags strings            Comma separated list of supported kube-apiserver flags, e.g: --request-timeout=2m0s,--kubelet-timeout=20s
		--block-size string                   Block size determines how many Pods can run per node vs total number of nodes per cluster (default "26")
		--container-runtime string            The container runtime for the cluster (default "containerd")
		--containers-cidr string              CIDR for container overlay (default "10.20.0.0/16")
		--controller-manager-flags strings    Comma separated list of supported kube-controller-manager flags, e.g: --large-cluster-size-threshold=60,--concurrent-statefulset-syncs=10
		--enable-kubeVirt                     Enables Kubernetes to run Virtual Machines within Pods. This feature is not recommended for production workloads, use either --enable-kubeVirt or --enable-kubeVirt=true to change
		--enable-profile-engine               Simplify cluster governance using the Platform9 Profile Engine, use either --enable-profile-engine or --enable-profile-engine=false to change (default true)
		--etcd-backup                         Enable automated etcd backups on this cluster, use either --etcd-backup or --etcd-backup=false to change (default true)
		--external-dns-name string            External DNS for master VIP
	-h, --help                                help for bootstrap
		--http-proxy string                   Specify the HTTP proxy for this cluster. Format-> <scheme>://<username>:<password>@<host>:<port>, username and password are optional.
		--interface-detction-method string    Interface detection method for Calico CNI (default "first-found")
	-i, --ip strings                          IP address of the host to be prepared
		--ip-encapsulation string             Encapsulates POD traffic in IP-in-IP between nodes (default "Always")
		--master-virtual-interface string     Physical interface for virtual IP association
		--master-virtual-ip string            Virtual IP address for cluster
		--metallb-ip-range string             Ip range for MetalLB
		--mfa string                          MFA token
		--monitoring                          Enable monitoring for this cluster, use either --monitoring or --monitoring=false to change (default true)
		--mtu-size string                     Maximum Transmission Unit (MTU) for the interface (default "1440")
		--nat int                             Packets destined outside the POD network will be SNAT'd using the node's IP (default 1)
		--network-plugin string               Specify network plugin ( Possible values: flannel or calico ) (default "calico")
		--network-plugin-operator             Will deploy Platform9 CRDs to enable multiple CNIs and features such as SR-IOV, use either --network-plugin-operator or --network-plugin-operator=true to change
		--network-stack int                   0 for ipv4 and 1 for ipv6
	-p, --password string                     Ssh password for the node (use 'single quotes' to pass password)
		--privileged                          Enable privileged mode for K8s API, use either --privileged or --privileged=false to change (default true)
	-r, --remove-existing-pkgs                Will remove previous installation if found, use either --remove-existing-pkgs or --remove-existing-pkgs=true to change
		--reserved-cpu string                 Comma separated list of CPUs to be reserved for the system, e.g: 4-8,9-12
		--scheduler-flags strings             Comma separated list of supported Kube-scheduler flags, e.g: --kube-api-burst=120,--log_file_max_size=3000
		--services-cidr string                CIDR for services overlay (default "10.21.0.0/16")
	-s, --ssh-key string                      Ssh key file for connecting to the node
	-e, --sudo-pass string                    Sudo password for user on remote host
		--tag string                          Add tag metadata to this cluster (key=value)
		--topology-manager-policy string      Topology manager policy (default "none")
		--use-hostname                        Use node hostname for cluster creation, use either --use-hostname or --use-hostname=true to change
	-u, --user string                         Ssh username for the node


Global Flags:
		--log-dir string   path to save logs
		--no-prompt        disable all user prompts
		--verbose          print verbose logs

```

```sh
#pf9ctl bootstrap testCluster --pmk-version 1.21.3-pmk.72
✓ Loaded Config Successfully
✓ Node is not onboarded and not attached to any cluster
✓ Removal of existing CLI
✓ Existing Platform9 Packages Check
✓ Required OS Packages Check
✓ SudoCheck
✓ CPUCheck
✓ DiskCheck
✓ MemoryCheck
✓ PortCheck
✓ Existing Kubernetes Cluster Check
✓ Check lock on dpkg
✓ Check lock on apt
✓ Check if system is booted with systemd
✓ Check time synchronization
✓ Check if firewalld service is not running
✓ Disabling swap and removing swap in fstab

✓ Completed Pre-Requisite Checks successfully

Prep local node as master node for kubernetes cluster (y/n): y
✓ Platform9 packages installed successfully
✓ Initialised host successfully
✓ Host successfully attached to the Platform9 control-plane
✓ Cluster creation started
✓ Host is connected
✓ Attached node to the cluster
✓ Bootstrap successfully finished
Cluster creation started....This may take a few minutes....Check the latest status in UI
```
