#
# MIT License
#
# Copyright (c) 2024 EASL
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

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
