apk add --no-cache openrc
apk add --no-cache util-linux
apk add --no-cache openssh

# might remove for benchmark
ln -s agetty /etc/init.d/agetty.ttyS0
echo ttyS0 > /etc/securetty
rc-update add agetty.ttyS0 default

# might remove for benchmark
echo "root:root" | chpasswd

# might remove for benchmark?
echo "nameserver 1.1.1.1" >>/etc/resolv.conf

addgroup -g 1000 -S agentUser && adduser -u 1000 -S agentUser -G agentUser

chown agentUser:agentUser /etc/init.d/agent
chmod u+x /etc/init.d/agent
chown agentUser:agentUser /usr/local/bin/agent
chmod u+x /usr/local/bin/agent

rc-update add devfs boot
rc-update add procfs boot
rc-update add sysfs boot

rc-update add agent boot

# for extracting logs with scp
# rc-update add sshd
# echo "PermitRootLogin yes" >> /etc/ssh/sshd_config


# Copy the configured system to the rootfs image
for d in bin etc lib root sbin usr; do tar c "/$d" | tar x -C /rootfs; done
for dir in dev proc run sys var tmp; do mkdir /rootfs/${dir}; done

chmod 1777 /rootfs/tmp
mkdir -p /rootfs/home/agentUser/
chown 1000:1000 /rootfs/home/agentUser/
