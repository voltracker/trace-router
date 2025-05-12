# Trace Router

App that samples outgoing traffic and runs a traceroute on the destination IP on random packets. The idea is to built a graph of all the hops that the traffic on your local network takes to
reach it's final destination.

In future I will likely look to implement a Berkley Packet Filter for sampling traffic, as this may be more efficient than just having a background thread running constantly. (Still need to research that tho)
