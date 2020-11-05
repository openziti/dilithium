# Dilithium Framework Concepts

The following are a collection of orthogonal concepts, which together comprise much of the core footprint of what the `dilithium` framework does. Understanding each individually, and then combined in aggregate will give a pretty clear picture of how `dilithium`-based protocols function.

## Directional txPortal/rxPortal Pair

![Directional txPortal/rxPortal Pair](images/concepts/directional_rxtx_pair.png)

The message-oriented components are typically described here as one half of a bi-directional communications implementation. A single `txPortal`&rarr;`rxPortal` pair manifests a single direction of communication. To realize a bi-directional communications link, a pair of `txPortal`&rarr;`rxPortal` components will be required, one for each direction.

The `txPortal` admits data as an array of octets from another software layer, which it then turns into messages that are transmitted across the message-passing infrastructure. The `rxPortal` receives messages from the message-passing infrastructure and reconstructs these as a stream of octets, which are delivered to its client.

The message-passing infrastructure is assumed to be unreliable.

## Rate Limiting Flow Control

![Rate Limiting Inputs](images/concepts/rate_limiting_inputs.png)

An important goal for these components is to simultaneously maximize the throughput of a message-passing system, while also limiting the flow such that the system is not overwhelmed. There's a delicate balance to be maintained, which must automatically adjust to changing backpressures and weather conditions.

`dilithium`-based protocols use a traditional _windowing_ model to manage the communication rate. In this framework, we call the window a _portal_ (because we're cheeky). The portal _capacity_ represents the total size of the data that is allowed to be present in the link, transiting between the `txPortal` and the `rxPortal` at the current point in time. the `txPortalSz` represents the amount of data managed in the transmitting side of the communication. The `rxPortalSz` represents the amount of data buffered in the receiving side.

The `txPortal` will only admit data from its client into the link when the minimum of the `(capacity - txPortalSz)` or `(capacity - rxPortalSz)` is equal to or larger than the size of the data the client wants to transmit. Until that amount of capacity becomes available, the `txPortal` will block its client, providing backpressure. 

## Loss Handling

![Loss Handling](images/concepts/loss_handling.png)

`dilithium`-based implementations operate over unreliable message-passing infrastructures (primarily, UDP datagrams across the internet). In order to turn these unreliable message-passing systems into a reliable stream of communications, `dilithium` must implement loss handling and mitigation.

Also, unreliable message-passing infrastructures usually do not guarantee message ordering, and `dilithium` must also handle these cases as well.

And, to keep it interesting... the control messages (`ACK`) can also go missing or arrive in unusual orders. `dilithium` must also operate correctly in these cases.

In the diagram above, we see the 3 typical types of communications that happen between the `txPortal`&rarr;`rxPortal` pair. The first type of messages are the _nominal transmission_ messages, which are portions of the stream data that have been assigned a _sequence identifier_. Typically when the `rxPortal` receives these messages it sends back an _acknowledgement_ (`ACK`) message, notifying the `txPortal` that it received the payload. If the `txPortal` does not receive an `ACK` within a timeout period (see section below on _Round-Trip Time Probes_), it will retransmit the payload again. It will continue retransmitting at the end of the expiry period, until it receives an `ACK` from the `rxPortal`.

In cases where the `ACK` message went missing, the retransmission mechanism will ultimately re-synchronize the state of the message between the `txPortal`&rarr;`rxPortal` pair.

## Retransmission Monitor

![Retransmission Monitor](images/concepts/retx_monitor.png)

Inside the `txPortal` the _retransmission monitor_ maintains a list of payloads, ordered by their retransmission deadlines. It runs in a loop, sending the next payload again whenever the payload's deadline is reached. Once a payload is retransmitted, the next deadline for retransmission is computed and the payload is appended to the end of the list.

Deadline computation is performed according to the `txPortal`'s observed _round-trip time_ (`rtt`).

## Round-Trip Time Probes

![Round-Trip Time Probes](images/concepts/rtt_probes.png)

A critical component in achieving high performance is retransmission of lost payloads with the correct timing. 

Payloads that are retransmitted to early are in danger of being an un-needed retransmission (the corresponding `ACK` may be in transit). Un-needed retransmissions consume bandwidth that could be used for productive data movement. 

Payloads that are retransmitted too late reduce the overall perceived performance of the link. When a retransmission is necessary it often fills in a "hole" in the `rxPortal` buffer like a Tetris piece, allowing a number of payloads to be released at once. The more payloads are pending, waiting on a retransmission, the more performance impact will be felt by late retransmissions.

In order to achieve the tightest retransmission timing possible, `dilithium` uses configurable `rtt` _probes_. Probing is accomplished by the `txPortal` capturing the current high-resolution wall clock time, and inserting that into an outbound payload. When the `rxPortal` receives a payload with an embedded `rtt` probe, it embeds that probe back into its outbound `ack` for that payload. When the `txPortal` receives an `ack` with an `rtt` probe embedded, it can compare that probe to the current wall clock time to determine how long it took to get a response to that payload from the other side.

`rtt` probes are sent according to a configurable period, and the resulting `rtt` value is averaged over a configurable number of previous times. That averaged value is then scaled using a configurable _scale_ value, and can have additional milliseconds of time added to it. That value is output as `retxMs`, which is the number of milliseconds that the `retx` monitor should use in computing retransmission deadlines.

## Portal Scaling

![Portal Scaling](images/concepts/portal_scaling.png)

The flow rate is controlled by the _capacity_ of the portal (window). Finding the ideal sending rate, which results in the most productive data transfer with the least unnecessary retransmission requires the continual adjustment of the portal capacity.

In `dilithium` protocols, there are currently 3 main factors which contribute to scaling the portal capacity value.

Successfully acked transmissions have the size of their productive payloads added to an _accumulator_. When the number of successfully acked transmissions reaches a configurable threshold value, the size of that accumulator is scaled according to a `scale` value and the result is added to the `capacity`. This is intended to be the primary mechanism that allows the window size to grow.

Duplicate acks (`dupack`) are counted. When the number of duplicate acks reaches a configurable threshold value, the portal capacity is multiplied by a configurable `scale` value.

Retransmissions (`retx`) are counted. When the number of retransmitted payloads reaches a configurable threshold value, the portal capacity is multiplied by a configurable `scale` value.

Whenever a counter reaches a threshold it is reset to zero. When `dupack` or `retx` counters reach their thresholds, in addition to multiplying the portal capacity by their `scale` values, they also multiply the successful transmission accumulator by a scale value (current implementations hardcode this scale to `0.0`), allowing them to clear or adjust the successful transmission accumulator.

## Profiles

![Profiles](images/concepts/profiles.png)

`dilithium` provides a mechanism for externally configuring all of the tunable parameters. Everything discussed previously in this concepts guide is tunable through the _profile_ mechanism.

When dealing with a pair of `txPortal`&rarr;`rxPortal` instances, there are effectively 2 profiles in use, one for each of the instances.

`dilithium` implementations can take advantages of asymmetric tunings, supporting asymmetric message-passing underlays. This means when implementing internet protocols on top of `dilithium`, a completely different profile can be used to tune the "upstream" versus the downstream.

As of version `0.3`, profile information is exchanged through the `HELLO` connection setup process. Future versions of `dilithium` will allow for dynamic profile exchange at any point in the lifecycle of a communications connection.

The current profile concept is a sealed set of knobs to tune the infrastructure for various operational and performance needs. Future iterations of the profile concept will likely expose extension points in the framework, allowing downstream code to participate in most of these framework concepts directly, chaning their behavior to even deeper degrees.

## Concepts in Progress

* Write Buffer (txPortal)
	+ Back-pressure
* Read Buffer (rxPortal)
	+ Ordering
	+ Back-pressure
* Extensible Framework
* Protocol Manifestations
	+ westworld3
	+ Ziti Transwarp