package util

import (
	"os"
	"path/filepath"
	"time"
)

var Files []string
var Pf9Packages []string
var RequiredPorts []string
var PortErr string
var ProcessesList []string //Kubernetes clusters processes list
var SwapOffDisabled bool   //If this is true the swapOff functionality will be disabled.
var SkipPrepNode bool
var HostDown bool
var EBSPermissions []string
var Route53Permissions []string
var EC2Permission []string
var VPCPermission []string
var IAMPermissions []string
var AutoScalingPermissions []string
var EKSPermissions []string
var GoogleCloudPermissions []string

var AzureContributorID string
var InstallerErrors = make(map[int]string)
var LogFileNamePath string

const (

	// number of CPUs
	MinCPUs = 2
	// RAM in GiBs
	MinMem = 12
	// Measure of a GiB in terms of bytes
	GB = 1024 * 1024
	// Disk size in GiBs
	MinDisk = 30
	// Disk size in GiBs
	MinAvailDisk = 15
	// Counter variable max value
	MaxLoopValue = 3
	//Attach Status Loop variable
	MaxRetryValue = 6

	CheckPass       = "PASS"
	CheckFail       = "FAIL"
	Invalid         = "Invalid"
	Valid           = "Valid"
	InvalidPassword = "Sorry, try again."

	PmkVersion = "1.20.11"
	Docker     = "docker"
	Calico     = "calico"
)

var (
	// Constants for check failure messages
	PyCliErr                = "Earlier version of pf9ctl already exists. This must be uninstalled."
	ExisitngInstallationErr = "Platform9 packages already exist. These must be uninstalled."
	SudoErr                 = "User running pf9ctl must have privilege (sudo) mode enabled."
	OSPackagesErr           = "Some OS packages needed for the CLI not found"
	CPUErr                  = "At least 2 CPUs are needed on host."
	DiskErr                 = "At least 30 GB of total disk space and 15 GB of free space is needed on host."
	MemErr                  = "At least 12 GB of memory is needed on host."
)

var (
	HomeDir, _ = os.UserHomeDir()
	// PyCliPath is the path of virtual env directory of the Python CLI
	PyCliPath = filepath.Join(HomeDir, "pf9/pf9-venv")
	// PyCliLink is the Symlink of the Python CLI
	PyCliLink      = "/usr/bin/pf9ctl"
	Centos         = "centos"
	Redhat         = "red hat"
	Ubuntu         = "ubuntu"
	CertsExpireErr = "certificate has expired or is not yet valid"

	//Pf9Dir is the base pf9dir
	Pf9Dir = filepath.Join(HomeDir, "pf9")
	//Pf9LogDir is the base path for creating log dir
	Pf9LogDir = filepath.Join(Pf9Dir, "log")
	// Pf9DBDir is the base dir for storing pf9 db config
	Pf9DBDir = filepath.Join(Pf9Dir, "db")
	// Pf9DBLoc represents location of the config file.
	Pf9DBLoc = filepath.Join(Pf9DBDir, "config.json")
	// Pf9Log represents location of the log.
	Pf9Log = filepath.Join(Pf9LogDir, "pf9ctl.log")
	// WaitPeriod is the sleep period for the cli
	// before it starts with the operations.
	WaitPeriod = time.Duration(60)

	OptDir = "/var/opt/pf9"

	//Location of ovf service file
	OVFLoc           = "/etc/systemd/system/ovf.service"
	VarDir           = "/var/log/pf9"
	EtcDir           = "/etc/pf9"
	DefaultPf9LogLoc = "pf9/log"
	Pf9LogLoc        string
	Pf9DirLoc        = filepath.Join(HomeDir, "/")

	//Auth,Dmesg,dpkg/yum files for Debian/Redhat
	DmesgLog = "/var/log/dmesg"
	MsgDeb   = "/var/log/syslog"
	LockDeb  = "/var/log/dpkg.log"
	MsgRed   = "/var/log/messages"
	LockRed  = "/var/log/yum.log"

	Confidential = []string{"--password", "--user-token"}
)

func init() {
	RequiredPorts = []string{"443", "2379", "2380", "8285", "10250", "10255", "4194", "3306", "8158", "5672", "5673", "8023", "9080", "6264", "5395", "8558"}
	ProcessesList = []string{"kubelet", "kube-proxy", "kube-apiserver", "kube-scheduler", "kube-controller", "etcd"}
	Pf9Packages = []string{"pf9-hostagent", "pf9-comms", "pf9-kube", "pf9-muster"}
	Files = []string{"/opt/cni", "/opt/containerd", "/var/lib/containerd", "/var/opt/pf9"}

	AzureContributorID = "b24988ac-6180-42a0-ab88-20f7382dd24c"

	GoogleCloudPermissions = []string{
		"roles/iam.serviceAccountUser",
		"roles/container.admin",
		"roles/compute.viewer",
		"roles/viewer",
	}

	EBSPermissions = []string{
		"elasticloadbalancing:AddTags",
		"elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
		"elasticloadbalancing:AttachLoadBalancerToSubnets",
		"elasticloadbalancing:ConfigureHealthCheck",
		"elasticloadbalancing:CreateLoadBalancer",
		"elasticloadbalancing:CreateLoadBalancerListeners",
		"elasticloadbalancing:DeleteLoadBalancer",
		"elasticloadbalancing:DescribeLoadBalancerAttributes",
		"elasticloadbalancing:DescribeLoadBalancers",
		"elasticloadbalancing:DescribeTags",
		"elasticloadbalancing:ModifyLoadBalancerAttributes",
		"elasticloadbalancing:RemoveTags",
	}

	Route53Permissions = []string{
		"route53:ChangeResourceRecordSets",
		"route53:GetChange",
		"route53:GetHostedZone",
		"route53:ListHostedZones",
		"route53:ListResourceRecordSets",
	}

	EC2Permission = []string{
		"ec2:AllocateAddress",
		"ec2:AssociateRouteTable",
		"ec2:AttachInternetGateway",
		"ec2:AuthorizeSecurityGroupEgress",
		"ec2:AuthorizeSecurityGroupIngress",
		"ec2:CreateInternetGateway",
		"ec2:CreateNatGateway",
		"ec2:CreateRoute",
		"ec2:CreateRouteTable",
		"ec2:CreateSecurityGroup",
		"ec2:CreateSubnet",
		"ec2:CreateTags",
		"ec2:DeleteInternetGateway",
		"ec2:DeleteNatGateway",
		"ec2:DeleteRoute",
		"ec2:DeleteRouteTable",
		"ec2:DeleteSecurityGroup",
		"ec2:DeleteSubnet",
		"ec2:DeleteTags",
		"ec2:DescribeAccountAttributes",
		"ec2:DescribeAddresses",
		"ec2:DescribeAvailabilityZones",
		"ec2:DescribeImages",
		"ec2:DescribeInstances",
		"ec2:DescribeInternetGateways",
		"ec2:DescribeKeyPairs",
		"ec2:DescribeNatGateways",
		"ec2:DescribeNetworkAcls",
		"ec2:DescribeNetworkInterfaces",
		"ec2:DescribeRegions",
		"ec2:DescribeRouteTables",
		"ec2:DescribeSecurityGroups",
		"ec2:DescribeSubnets",
		"ec2:DetachInternetGateway",
		"ec2:DisassociateRouteTable",
		"ec2:ImportKeyPair",
		"ec2:ModifySubnetAttribute",
		"ec2:ReleaseAddress",
		"ec2:ReplaceRouteTableAssociation",
		"ec2:RevokeSecurityGroupEgress",
		"ec2:RevokeSecurityGroupIngress",
		"ec2:RunInstances",
		"ec2:TerminateInstances",
	}

	VPCPermission = []string{
		"ec2:CreateVpc",
		"ec2:DeleteVpc",
		"ec2:DescribeVpcAttribute",
		"ec2:DescribeVpcClassicLink",
		"ec2:DescribeVpcClassicLinkDnsSupport",
		"ec2:DescribeVpcs",
		"ec2:ModifyVpcAttribute",
	}

	IAMPermissions = []string{
		"iam:AddRoleToInstanceProfile",
		"iam:CreateInstanceProfile",
		"iam:CreateRole",
		"iam:CreateServiceLinkedRole",
		"iam:DeleteInstanceProfile",
		"iam:DeleteRole",
		"iam:DeleteRolePolicy",
		"iam:GetInstanceProfile",
		"iam:GetRole",
		"iam:GetRolePolicy",
		"iam:GetUser",
		"iam:ListAttachedRolePolicies",
		"iam:ListInstanceProfilesForRole",
		"iam:ListRolePolicies",
		"iam:PassRole",
		"iam:PutRolePolicy",
		"iam:RemoveRoleFromInstanceProfile",
	}

	AutoScalingPermissions = []string{
		"autoscaling:AttachLoadBalancers",
		"autoscaling:CreateAutoScalingGroup",
		"autoscaling:CreateLaunchConfiguration",
		"autoscaling:CreateOrUpdateTags",
		"autoscaling:DeleteAutoScalingGroup",
		"autoscaling:DeleteLaunchConfiguration",
		"autoscaling:DeleteTags",
		"autoscaling:DescribeAutoScalingGroups",
		"autoscaling:DescribeLaunchConfigurations",
		"autoscaling:DescribeLoadBalancers",
		"autoscaling:DescribeScalingActivities",
		"autoscaling:DetachLoadBalancers",
		"autoscaling:EnableMetricsCollection",
		"autoscaling:UpdateAutoScalingGroup",
		"autoscaling:SuspendProcesses",
		"autoscaling:ResumeProcesses",
		"elasticloadbalancing:DescribeInstanceHealth",
	}

	EKSPermissions = []string{
		"eks:ListClusters",
		"eks:ListNodegroups",
		"eks:DescribeCluster",
		"eks:DescribeNodegroup",
		"eks:ListTagsForResource",
	}

	InstallerErrors[31] = "CONTROLLER_ADRESS_MISSING"
	InstallerErrors[32] = "CONTROLLER_USER_MISSING"
	InstallerErrors[33] = "CONTROLLER_PROJECTNAME_MISSING"
	InstallerErrors[34] = "CONTROLLER_PASSWORD_MISSING"
	InstallerErrors[35] = "KEYSTONE_REQUEST_FAILED"
	InstallerErrors[36] = "KEYSTONE_TOKEN_MISSING"
	InstallerErrors[41] = "NTPD_INSTALL_FAILED"
	InstallerErrors[42] = "NTPD_FAILED_TO_START"
	InstallerErrors[43] = "CHRONY_INSTALL_FAILED"
	InstallerErrors[44] = "CHRONY_FAILED_TO_START"
	InstallerErrors[51] = "ARCHITECTURE_NOT_SUPPORTED"
	InstallerErrors[51] = "OS_NOT_SUPPORTED"
	InstallerErrors[53] = "CORRUPT_SUDOERS_FILE"
	InstallerErrors[54] = "IMPORTANT_PORT_OCCUPIED"
	InstallerErrors[55] = "CONNECTION_FAILED"
	InstallerErrors[56] = "CONNECTION_FAILED_VIA_PROXY"
	InstallerErrors[57] = "PKG_MANAGER_MISSING"
	InstallerErrors[58] = "PF9_PACKAGES_PRESENT"
	InstallerErrors[61] = "HOSTAGENT_DEPENDANCY_INSTALLATION_FAILED"
	InstallerErrors[71] = "DOWNLOAD_NOCERT_ARGUMENT_MISSING"
	InstallerErrors[72] = "PACKAGE_LIST_DOWNLOAD_FAILED"
	InstallerErrors[73] = "PACKAGE_DOWNLOAD_FAILED"
	InstallerErrors[81] = "HOSTAGENT_PKG_INSTALLATION_FAILED"
	InstallerErrors[90] = "PROXY_SETUP_FAILED"
	InstallerErrors[100] = "UPDATE_CONFIG_FAILED"
	InstallerErrors[111] = "VOUCH_ARGUMENT_MISSING"
	InstallerErrors[112] = "HOST_CERTS_SCRIPT_FAILED"
	InstallerErrors[121] = "VMWARE_INSALL_FAILED"
	InstallerErrors[131] = "SYSTEMCTL_FAILED_TO_START_COMMS"
	InstallerErrors[132] = "COMMS_NOT_UP"
	InstallerErrors[133] = "SYSTEMCTL_FAILED_TO_START_SIDEKICK"
	InstallerErrors[134] = "SIDEKICK_NOT_UP"
	InstallerErrors[135] = "SYSTEMCTL_FAILED_TO_START_HOSTAGENT"
	InstallerErrors[136] = "HOSTAGENT_NOT_UP"
	InstallerErrors[137] = "HOSTAGENT_NOT_RUNNING"
	InstallerErrors[138] = "HOSTAGENT_FILES_MISSING"
	InstallerErrors[139] = "COMMS_NOT_RUNNING"
	InstallerErrors[141] = "PARAMS_MISSING"
	InstallerErrors[142] = "CREDS_NOT_NEEDED"

}

//These are the constants needed for everything version related
const (
	Version         string = "pf9ctl version: v1.15"
	AWSBucketName   string = "pmkft-assets"
	AWSBucketKey    string = "pf9ctl"
	AWSBucketRegion string = "us-west-1"
	BucketPath      string = "https://" + AWSBucketName + ".s3." + AWSBucketRegion + ".amazonaws.com/" + AWSBucketKey + "_setup"
)
