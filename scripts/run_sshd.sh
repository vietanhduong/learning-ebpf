#!/bin/bash
set -e

function check_env_set() {
  if [[ -z "${!1}" ]]; then
    echo "The environment variable \"$1\" needs to be set"
    exit 1
  fi
}

source /test_fs/ssh_env.sh

check_env_set SSH_PUB_KEY

mkdir -p "/etc/ssh"
echo "PermitRootLogin without-password" >>/etc/ssh/sshd_config
echo "PrintLastLog no" >>/etc/ssh/sshd_config
mkdir -p "/root/.ssh"
cat "${SSH_PUB_KEY}" >>/root/.ssh/authorized_keys
ssh-keygen -A -N '' -b 1024

useradd sshd
mkdir -p "/run/sshd"
mkdir -p "/var/log"

echo "Starting SSH Daemon"
exec /usr/sbin/sshd -p 22 -e -D
