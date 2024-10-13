#!/bin/bash

function test_exec {
    exec_name=$1
    cp /bin/echo /usr/local/bin/${exec_name}
    ${exec_name} hello
}

echo "For: chkconfig Execution"
chkconfig

echo "For: Cryptocurrency Mining Process Execution"
test_exec xmr-stak-cpu

echo "For: Compiler Tool Execution"
test_exec make

echo "For: crontab Execution"
test_exec crontab

echo "For: iptables Execution"
test_exec iptables

echo "For: Linux Group Add Execution"
echo "And: Password Binaries"
echo "And: Shadow File Modification"
groupadd spud

echo "For: Linux User Add Execution"
useradd spudnik

echo "For: Login Binaries"
login -V

echo "For: Netcat Execution Detected"
test_exec nc

echo "For: Network Management Execution"
test_exec ethtool

echo "For: nmap Execution"
test_exec nmap

echo "For: Process Targeting Cluster Kubelet Endpoint"
curl http://127.0.0.1:10248/

echo "For: Process Targeting Cluster Kubernetes Docker Stats Endpoint"
curl http://127.0.0.1:4194/

echo "For: Process Targeting Kubernetes Service Endpoint"
curl https://127.0.0.1/apis/role

echo "Everything triggers: Process with UID 0"

echo "For: Secure Shell Server (sshd) Execution"
echo "And: SetUID Processes"
test_exec sshd

echo "For: Shell Spawned by Java Application"
cp /bin/bash /bin/java
# Uses csh and a separate file to avoid running as a coreutils-prog-shebang process
/bin/java -x run-csh.sh

echo "For: Red Hat Package Manager Execution"
yum --version

echo "For: Remote File Copy Binary Execution"
test_exec scp

echo "For: systemctl Execution"
systemctl --version

echo "For: systemd Execution"
test_exec systemd

echo "For: Ubuntu Package Manager Execution"
test_exec apt

echo "For: Wget Execution"
test_exec wget

sleep 36000
