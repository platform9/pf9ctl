#!/bin/bash

set -o pipefail

start_time=$(date +"%s.%N")

assert() {
    if [ $# -gt 0 ]; then stdout_log "ASSERT: ${1}"; fi
    if [[ -f ${log_file} ]]; then
        echo -e "\n\n"
	echo ""
	echo "Installation failed, Here are the last 10 lines from the log"
	echo "The full installation log is available at ${log_file}"
	echo "If more information is needed re-run the install with --debug"
	echo "$(tail ${log_file})"
    else
	echo "Installation failed prior to log file being created"
	echo "Try re-running with --debug"
	echo "Installation instructions: https://docs.platform9.com/kubernetes/PMK-CLI/#installation"
    fi
    exit 1
}

debugging() {
    # This function handles formatting all debugging text.
    # debugging is always sent to the logfile.
    # If no debug flag, this function will silently log debugging messages.
    # If debug flag is present then debug output will be formatted then echo'd to stdout and sent to logfile.
    # If debug flag is present messages sent to stdout_log will be forwarded to debugging for consistancy.

    # Avoid error if bc is not installed yet
    if (which bc > /dev/null 2>&1); then
	output="$(date +"%T")$(bc <<<$(date +"%s.%N")-${start_time}) :$(basename $0) : ${1}"
    else
	output="$(date +"%T"):$(basename $0) : ${1}"
    fi

    if [ -f "${log_file}" ]; then
	echo "${output}" 2>&1 >> ${log_file}
    fi
    echo "${output}"
}

stdout_log() {
    echo "$1"
    debugging "$1"
}

initialize_basedir() {
    debugging "Initializing: ${pf9_basedir}"
    for dir in ${pf9_state_dirs}; do
	debugging "Ensuring ${dir} exists"
        if ! mkdir -p "${dir}" > /dev/null 2>&1; then assert "Failed to create directory: ${dir}"; fi
    done
    debugging "Ensuring ${log_file} exists"
    if ! mkdir -p "${dir}" > /dev/null 2>&1; then assert "Failed to create directory: ${dir}"; fi
    if ! touch "${log_file}" > /dev/null 2>&1; then assert "failed to create log file: ${log_file}"; fi
}

refresh_symlink() {
    # Create symlink in /usr/bin
    if [ -L /usr/bin/pf9ctl ]; then
	if ! (sudo rm /usr/bin/pf9ctl >> ${log_file} 2>&1); then
		assert "Failed to remove existing symlink: ${cli_exec}"; fi
	fi
    if ! (sudo ln -s ${cli_exec} /usr/bin/pf9ctl >> ${log_file} 2>&1); then
	    assert "Failed to create Platform9 CLI symlink in /usr/bin"; fi
}

check_installation() {
    if ! (${cli_exec} --help >> ${log_file} 2>&1); then
	assert "Installation of Platform9 CLI Failed"; fi
}

download_cli_binary() {
	sudo rm ${cli_exec}
	cd ${pf9_bin} && sudo curl -O ${cli_path} -o ${log_file} 2>&1 && cd -
	sudo chmod 555 ${cli_exec}
}

##main

echo ""
stdout_log "** SUDO REQUIRED FOR ALL COMMANDS **"
stdout_log "Why is SUDO required?: The CLI performs operations like installing packages and configuring services. These require SUDO privileges to run successfully."
echo ""

pf9_basedir=$(dirname ~/pf9/.)
log_file=${pf9_basedir}/log/cli_install.log
pf9_bin=${pf9_basedir}/bin
pf9_state_dirs="${pf9_bin} ${pf9_basedir}/db ${pf9_basedir}/log"
cli_exec=${pf9_bin}/pf9ctl
cli_path="https://pmkft-assets.s3-us-west-1.amazonaws.com/pf9ctl"

initialize_basedir
download_cli_binary
refresh_symlink
check_installation

echo ""
stdout_log "Platform9 CLI installation completed successfully"
echo "To start building a Kubernetes cluster execute:"
echo "        ${cli_exec} help"
echo ""
eval "${cli_exec}" help
