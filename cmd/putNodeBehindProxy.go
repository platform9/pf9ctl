package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	proxySetting objects.ProxySetting
	noProxy      = "localhost,127.0.0.1,::1,localhost.localdomain,localhost4,localhost6"
	noProxyList  string
)

var putNodeBehindProxycmd = &cobra.Command{
	Use:     "set-proxy",
	Short:   "Put existing pmk node behind proxy",
	Example: "pf9ctl set-proxy --protocol <http/https> --host-ip <proxyIP> --port <proxyPort> --proxy-user <proxyUsername> --proxy-pass <proxyPassword>",
	Run:     putNodeBehindProxyRun,
}

func init() {
	putNodeBehindProxycmd.Flags().StringVar(&proxySetting.Proxy.Protocol, "protocol", "", "Proxy protocol")
	putNodeBehindProxycmd.Flags().StringVar(&proxySetting.Proxy.Host, "host-ip", "", "Proxy IP")
	putNodeBehindProxycmd.Flags().StringVar(&proxySetting.Proxy.Port, "port", "", "Proxy port")
	putNodeBehindProxycmd.Flags().StringVar(&proxySetting.Proxy.User, "proxy-user", "", "Proxy username")
	putNodeBehindProxycmd.Flags().StringVar(&proxySetting.Proxy.Pass, "proxy-password", "", "Proxy password")
	putNodeBehindProxycmd.Flags().StringVar(&noProxyList, "no-proxy", "", "comma seperated list of IPs or domain names")
	putNodeBehindProxycmd.MarkFlagRequired("protocol")
	putNodeBehindProxycmd.MarkFlagRequired("host-ip")
	putNodeBehindProxycmd.MarkFlagRequired("port")

	//Remote node details
	putNodeBehindProxycmd.Flags().StringVarP(&nodeConfig.User, "host-user", "u", "", "ssh username for the node")
	putNodeBehindProxycmd.Flags().StringVarP(&nodeConfig.Password, "host-password", "p", "", "ssh password for the node (use 'single quotes' to pass password)")
	putNodeBehindProxycmd.Flags().StringVarP(&nodeConfig.SshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	putNodeBehindProxycmd.Flags().StringSliceVarP(&nodeConfig.IPs, "ip", "i", []string{}, "ssh Ip of host")
	rootCmd.AddCommand(putNodeBehindProxycmd)
}

func putNodeBehindProxyRun(cmd *cobra.Command, args []string) {
	zap.S().Debugf("Setting proxy on host")
	var proxy_url string
	if proxySetting.Proxy.User != "" && proxySetting.Proxy.Pass != "" {
		proxy_url = fmt.Sprintf("%s://%s:%s@%s:%s", proxySetting.Proxy.Protocol, proxySetting.Proxy.User, proxySetting.Proxy.Pass, proxySetting.Proxy.Host, proxySetting.Proxy.Port)
	} else {
		proxy_url = fmt.Sprintf("%s://%s:%s", proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port)
	}

	commsProxyFilePath := "/etc/pf9/comms_proxy_cfg.json"
	hostAgentEnvFile := "/opt/pf9/hostagent/pf9-hostagent.env"

	var envs = "export http_proxy=" + proxy_url + "\n" +
		"export https_proxy=" + proxy_url + "\n" +
		"export HTTP_PROXY=" + proxy_url + "\n" +
		"export HTTPS_PROXY=" + proxy_url + "\n" +
		"export no_proxy=" + noProxy + "\n" +
		"export NO_PROXY=" + noProxy

	detachedMode := cmd.Flags().Changed("no-prompt")

	if cmdexec.CheckRemote(nodeConfig) {
		if !config.ValidateNodeConfig(&nodeConfig, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	executor, err := cmdexec.GetExecutor("", nodeConfig)
	if err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	//If node is already onboarded this /opt/pf9/hostagent/pf9-hostagent.env file will present bydefault
	//Append pf9-hostagent proxy settings

	//Handle rerun
	cmnd := fmt.Sprintf("grep http_proxy %s", hostAgentEnvFile)
	_, err = executor.RunWithStdout("bash", "-c", cmnd)
	if err == nil {
		//Remove existing proxy settings
		zap.S().Debugf("Removing existing proxy envs")
		//Move required lines to temp file
		cmnd = fmt.Sprintf("grep -iv _proxy %s > %s.tmp", hostAgentEnvFile, hostAgentEnvFile)
		_, err = executor.RunWithStdout("bash", "-c", cmnd)
		if err != nil {
			zap.S().Fatalf("Unable to remove existing proxy from %s ", hostAgentEnvFile)
		}
		//Move temp file back to original file
		cmnd = fmt.Sprintf("mv %s{.tmp,}", hostAgentEnvFile)
		_, err = executor.RunWithStdout("bash", "-c", cmnd)
		if err != nil {
			zap.S().Fatalf("Failed while moving temp file back to original file %s ", hostAgentEnvFile)
		}
		//Remove temp file
		cmnd = fmt.Sprintf("rm -rf /opt/pf9/hostagent/pf9-hostagent.env.tmp")
		_, err = executor.RunWithStdout("bash", "-c", cmnd)
		if err != nil {
			zap.S().Debugf("File %s.tmp not removed", hostAgentEnvFile)
		}
	}

	cmnd = fmt.Sprintf("ls %s", hostAgentEnvFile)
	_, err = executor.RunWithStdout("bash", "-c", cmnd)
	if err != nil {
		zap.S().Fatalf("HostAgentEnv %s file is not present", hostAgentEnvFile)
	}

	zap.S().Infof("Adding proxy setting to %s", hostAgentEnvFile)
	cmnd = fmt.Sprintf(`tee -a %s >> /dev/null <<EOT 
%s 
EOT`, hostAgentEnvFile, envs)

	_, err = executor.RunWithStdout("bash", "-c", cmnd)
	if err != nil {
		zap.S().Fatalf("Unable to add proxy setting to %s ", hostAgentEnvFile)
	} else {
		zap.S().Infof("pf9-hostagent proxy setting added to %s ", hostAgentEnvFile)
	}

	zap.S().Infof("Adding proxy setting to %s ", commsProxyFilePath)
	//write pf9-comms proxy setting to /etc/pf9/comms_proxy_cfg.json
	_, err = executor.RunWithStdout("bash", "-c", "touch /etc/pf9/comms_proxy_cfg.json")
	if err != nil {
		zap.S().Fatalf("Unable to create %s file", commsProxyFilePath)
	}

	var json string
	if proxySetting.Proxy.User != "" && proxySetting.Proxy.Pass != "" && noProxyList != "" {
		json = fmt.Sprintf(`{"http_proxy":{"protocol":"%s", "host":"%s", "port":%s, "user":"%s", "pass":"%s","no_proxy":"%s"}}`, proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port, proxySetting.Proxy.User, proxySetting.Proxy.Pass, noProxyList)
	} else if noProxyList != "" {
		json = fmt.Sprintf(`{"http_proxy":{"protocol":"%s", "host":"%s", "port":%s,"no_proxy":"%s"}}`, proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port, noProxyList)
	} else {
		json = fmt.Sprintf(`{"http_proxy":{"protocol":"%s", "host":"%s", "port":%s}}`, proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port)
	}

	isRemote := cmdexec.CheckRemote(nodeConfig)

	if isRemote {
		if proxySetting.Proxy.User != "" && proxySetting.Proxy.Pass != "" && noProxyList != "" {
			json = fmt.Sprintf(`{\"http_proxy\":{\"protocol\":\"%s\", \"host\":\"%s\", \"port\":%s, \"user\":\"%s\", \"pass\":\"%s\", \"no_proxy\":\"%s\"}}`, proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port, proxySetting.Proxy.User, proxySetting.Proxy.Pass, noProxyList)
		} else if noProxyList != "" {
			json = fmt.Sprintf(`{\"http_proxy\":{\"protocol\":\"%s\", \"host\":\"%s\", \"port\":%s, \"no_proxy\":\"%s\"}}`, proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port, noProxyList)
		} else {
			json = fmt.Sprintf(`{\"http_proxy\":{\"protocol\":\"%s\", \"host\":\"%s\", \"port\":%s}}`, proxySetting.Proxy.Protocol, proxySetting.Proxy.Host, proxySetting.Proxy.Port)
		}
	}

	cmnd = fmt.Sprintf(`echo '%s' 2>&1 | tee %s`, json, commsProxyFilePath)

	_, err = executor.RunWithStdout("bash", "-c", cmnd)
	if err != nil {
		zap.S().Fatalf("Unable to add proxy setting to %s file", commsProxyFilePath)
	} else {
		zap.S().Infof("pf9-comms proxy settng added to %s ", commsProxyFilePath)
	}

	//change file ownership to pf9 group
	cmnd = fmt.Sprintf("chown pf9:pf9group %s", commsProxyFilePath)
	_, err = executor.RunWithStdout("bash", "-c", cmnd)
	if err != nil {
		zap.S().Errorf("Unable to change ownership of %s file", commsProxyFilePath)
	} else {
		zap.S().Infof("Changed ownership of %s file to pf9:pf9group", commsProxyFilePath)
	}

	//Restart pf9 services
	zap.S().Info("Restarting Platform9 services")
	_, err = executor.RunWithStdout("bash", "-c", "systemctl restart pf9-hostagent")
	if err != nil {
		zap.S().Fatalf("Unable to restart pf9-hostagent")
	} else {
		zap.S().Infof("pf9-hostagent is restarted")
	}

	_, err = executor.RunWithStdout("bash", "-c", "systemctl restart pf9-comms")
	if err != nil {
		zap.S().Fatalf("Unable to restart pf9-comms")
	} else {
		zap.S().Infof("pf9-comms is restarted")
	}

}
