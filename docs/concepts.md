# Dilithium Framework Concepts

## Directional txPortal/rxPortal Pair

![Directional txPortal/rxPortal Pair](images/directional_rxtx_pair.png)

The message-oriented components are typically described here as one half of a bi-directional communications implementation. A single `txPortal` &rarr; `rxPortal` pair manifests a single direction of communication. To realize a bi-directional communications link, two pairs of `txPortal`/`rxPortal` components will be required, one for each direction.

## Concepts in Progress

* Rate Limiting Flow Control
	+ Portal Size (capacity/txPortalSz/rxPortalSz)
	+ Capacity
* Loss Handling
	+ Acknowledgement (ack)
	+ Retransmission (retx)
	+ Retransmission Monitor
* Round-Trip Time Probes
* Portal Scaling
	+ Successful Transmission
	+ Duplicate Acknowledgement
	+ Retransmission
* Profiles
* Protocol Manifestations
	+ westworld3
* Write Buffer (txPortal)
	+ Back-pressure
* Read Buffer (rxPortal)
	+ Ordering
	+ Back-pressure
* Extensible Framework
	+ Ziti Transwarp