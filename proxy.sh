#!/bin/bash
# https://linkerd.io/2.11/reference/iptables/#:~:text=redirect-all-outgoing%3A%20the%20last%20rule%20in%20the%20chain%2C%20it,it%20has%20not%20been%20produced%20by%20the%20proxy.

### Inbound
# create custom user chain
iptables -t nat -N PROXY_INIT_REDIRECT

# send all incoming packets to the custom chain
iptables -t nat -A PREROUTING -j PROXY_INIT_REDIRECT

# ignore packets whose destination ports are in the list
# ignoring requests going out to epwatcher
iptables -t nat -A PROXY_INIT_REDIRECT -p tcp --match multiport --dports 30000 -j RETURN

# configure iptables to redirect everything to port 62081
iptables -t nat -A PROXY_INIT_REDIRECT -p tcp -j REDIRECT --to-port 62081

### Outbound
# create a custom chain
iptables -t nat -N PROXY_INIT_OUTPUT

# send all outbound packets to the custom chain
iptables -t nat -A OUTPUT -j PROXY_INIT_OUTPUT

# skip packets owned by proxy
iptables -t nat -A PROXY_INIT_OUTPUT -m owner --uid-owner 2102 -j RETURN

# skip packets sent over the loopback interface
iptables -t nat -A PROXY_INIT_OUTPUT -o lo -j RETURN

# ignore destination ports in the list
# iptables -t nat -A PROXY_INIT_OUTPUT -p tcp --match multiport --dports <ports> -j RETURN

# redirect all outbound packets to the proxy's outbound port
iptables -t nat -A PROXY_INIT_OUTPUT -p tcp -j REDIRECT --to-port 62082