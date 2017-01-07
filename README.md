Go ToyVpn
============

implement ToyVpn in native golang shared libraries.


## Different between original ToyVpn
1. call native golang shared library
2. go through TCP, not UDP

## Build
for apk:
> $ ./gradlew build --info


for server:
> $ cd server/golang/
> $ sh build.sh


## Run

### 1. setting TUN interface

```
# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

# Pick a range of private addresses and perform NAT over eth0.
iptables -t nat -A POSTROUTING -s 10.0.0.0/8 -o eth0 -j MASQUERADE

# Create a TUN interface.
tunctl -n -t tun2

# Set the addresses and bring up the interface.
ifconfig tun2 10.0.0.0/8 up
```


### 2. start server

```
# Create a server on port 23456 with shared secret "test123456".
cd server/golang/
./server -bind ":23456" -tun tun2 -m 1400 -s test123456
```

### 3. connect to server by app


## TODO
- [ ] fix buggy Android log
- [ ] show current status and notification


