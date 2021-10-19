# pf9ctl

### Status
![Go](https://github.com/roopakparikh/pf9ctl/workflows/Go/badge.svg)

### Purpose
   CLI tool for Platform9 management. This is under heavy development, please use with care
   
### Requirements(Prerequisites)

- CPUs: Minimum 2 CPUs needed on host
- RAM: 12 GB 
- Disk: At least 30 GB of total disk space and 15 GB of free space is needed on host
- Sudo access to the user
- OS(Supported) : 
    - Ubuntu (16.04,18.04,20.04)
    - Centos (7.x ,8.3)

### Proxy support

The CLI allows configuration where all HTTPS requests can be routed through a proxy. See the `Configuration` section to see how to configure the proxy URL.

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
  attach-node Attaches node to k8s cluster
  bundle      Gathers support bundle and uploads to S3
  check-node  Check prerequisites for k8s
  config      Create or get config
  help        Help about any command
  prep-node   set up prerequisites & prep the node for k8s
  version     Current version of CLI being used

Flags:
  -h, --help      help for pf9ctl
      --verbose   print verbose logs

Use "pf9ctl [command] --help" for more information about a command.
```
- **Version**

  **This command is used to get the current version of the CLI**
```sh
#pf9ctl version

pf9ctl version: v1.8

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
#pf9ctl upgrade --skipCheck
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
  -u, --account_url string   sets account_url
  -h, --help                 help for set
  -p, --password string      sets password (use 'single quotes' to pass password)
  -l, --proxy_url string     sets proxy URL, can be specified as [<protocol>][<username>:<password>@]<host>:<port>
  -r, --region string        sets region
  -t, --tenant string        sets tenant
  -e, --username string      sets username

Global Flags:
      --verbose   print verbose logs
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
  -h, --help              help for check-node
  -i, --ip strings        IP address of host to be prepared
  -p, --password string   ssh password for the nodes
  -s, --ssh-key string    ssh key file for connecting to the nodes
  -u, --user string       ssh username for the nodes

Global Flags:
      --verbose   print verbose logs
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
  -h, --help              help for prep-node
  -i, --ip strings        IP address of host to be prepared
  -p, --password string   ssh password for the nodes
  -c, --skipChecks        Will skip optional checks if true
  -s, --ssh-key string    ssh key file for connecting to the nodes
  -u, --user string       ssh username for the nodes

Global Flags:
      --verbose   print verbose logs

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
  -h, --help              help for bundle
  -i, --ip strings        IP address of host to be prepared
  -p, --password string   ssh password for the nodes
  -s, --ssh-key string    ssh key file for connecting to the nodes
  -u, --user string       ssh username for the nodes

Global Flags:
      --verbose   print verbose logs
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
  -w, --worker-ip strings   worker node ip address

Global Flags:
      --verbose   print verbose logs
```


```sh
#pf9ctl attach-node -m 172.20.7.66 -w 172.20.7.58 test-cluster
✓ Loaded Config Successfully
2021-05-26T11:58:01.9579Z	INFO	Worker node(s) [bf5364cf-e2fd-4500-97fb-0b01be26084f] attached to cluster
2021-05-26T11:58:03.6328Z	INFO	Master node(s) [615c1042-48a3-42e8-8003-ac135d12e6f4] attached to cluster
```
  
  **bootstrap**

```sh
#pf9ctl bootstrap --help

Bootstrap a single node Kubernetes cluster with your current
	host as the Kubernetes node. Read more at
	http://pf9.io/cli_clbootstrap.

Usage:
  pf9ctl bootstrap [flags]

Flags:
      --allowWorkloadsOnMaster   Taint master nodes ( to enable workloads ) (default true)
      --appCatalogEnabled        Enable Helm application catalog
      --containersCidr string    CIDR for container overlay (default "10.20.0.0/16")
      --externalDnsName string   External DNS for master VIP
  -h, --help                     help for bootstrap
      --masterVip string         IP Address for VIP for master nodes
      --masterVipIf string       Interface name for master / worker nodes
      --metallbIpRange string    Ip range for MetalLB
      --networkPlugin string     Specify network plugin ( Possible values: flannel or calico ) (default "flannel")
      --privileged               Enable privileged mode for K8s API. Default: true (default true)
      --servicesCidr string      CIDR for services overlay (default "10.21.0.0/16")

Global Flags:
      --verbose   print verbose logs
```

```sh
#pf9ctl bootstrap singletestcluster2
2021-10-19T12:24:48.7484Z	INFO	Running Checks for Bootstrap Command
✓ Loaded Config Successfully
2021-10-19T12:24:50.3744Z	INFO	Creating the cluster singletestcluster2
2021-10-19T12:25:55.5814Z	INFO	Attaching node to the cluster singletestcluster2
2021-10-19T12:25:56.3007Z	INFO	=======Bootstrap successfully finished========
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