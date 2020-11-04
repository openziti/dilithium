# Dilithium Framework Concepts

## Directional txPortal/rxPortal Pair

![Directional txPortal/rxPortal Pair](images/directional_rxtx_pair.png)

The message-oriented components are typically described here as one half of a bi-directional communications implementation. A single `txPortal` &rarr; `rxPortal` pair manifests a single direction of communication. To realize a bi-directional communications link, two pairs of `txPortal`/`rxPorta` components will be required, one for each direction.

The `txPortal` admits data as an array of octets from another software layer, which it then turns into messages that are transmitted across the message-passing infrastructure. The `rxPortal` receives messages from the message-passing infrastructure and reconstructs these as a stream of octets, which are delivered to its client.

The message-passing infrastructure is assumed to be unreliable.

## Rate Limiting Flow Control

![Rate Limiting Inputs](images/rate_limiting_inputs.png)

An important goal for these components is to simultaneously maximize the throughput of a message-passing system, while also limiting the flow such that the system is not overwhelmed. There's a delicate balance to be maintained, which must automatically adjust to changing backpressures and weather conditions.

`dilithium`-based protocols use a traditional _windowing_ model to manage the communication rate. In this framework, we call the window a _portal_ (because we're cheeky). The portal _capacity_ represents the total size of the data that is allowed to be present in the link, transiting between the `txPortal` and the `rxPortal` at the current point in time. the `txPortalSz` represents the amount of data managed in the transmitting side of the communication. The `rxPortalSz` represents the amount of data buffered in the receiving side.

The `txPortal` will only admit data from its client into the link when the minimum of the `(capacity - txPortalSz)` or `(capacity - rxPortalSz)` is equal to or larger than the size of the data the client wants to transmit. Until that amount of capacity becomes available, the `txPortal` will block its client, providing backpressure. 

## Rx/Tx Components Overview

![Rx/Tx Components Overview](images/rxtx_components.png)

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