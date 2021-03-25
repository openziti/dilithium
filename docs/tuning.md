# Westworld3 Tuning Guide

This guide explains the inner workings of the `westworld3` protocol along with the `westworld3` profile parameters.

This is what a complete `westworld3.Profile` baseline definition looks like:

```
*westworld3.Profile {
	randomize_seq                   false
	connection_setup_timeout_ms     5000
	connection_inactive_timeout_ms  15000
	send_keepalive                  true
	close_wait_ms                   5000
	close_check_ms                  500
	tx_portal_start_sz              98304
	tx_portal_min_sz                16384
	tx_portal_max_sz                4194304
	tx_portal_increase_thresh       224
	tx_portal_increase_scale        1
	tx_portal_dupack_thresh         64
	tx_portal_dupack_capacity_scale 0.9
	tx_portal_dupack_success_scale  0.75
	tx_portal_retx_thresh           64
	tx_portal_retx_capacity_scale   0.75
	tx_portal_retx_success_scale    0.825
	tx_portal_rx_sz_pressure_scale  2.8911
	retx_start_ms                   200
	retx_scale                      1.5
	retx_scale_floor                1
	retx_add_ms                     0
	retx_evaluation_ms              2000
	retx_evaluation_scale_incr      0.15
	retx_evaluation_scale_decr      0.01
	retx_batch_ms                   2
	rtt_probe_ms                    50
	rtt_probe_avg                   8
	rx_portal_sz_pacing_thresh      0.5
	max_segment_sz                  1450
	pool_buffer_sz                  65536
	rx_buffer_sz                    16777216
	tx_buffer_sz                    16777216
	tx_portal_tree_len              16384
	retx_monitor_tree_len           65536
	rx_portal_tree_len              16384
	listener_peers_tree_len         1024
	reads_queue_len                 1024
	listener_rx_queue_len           1024
	accept_queue_len                1024
}
```

## randomize_seq

If the `randomize_seq` switch is set to `false` (the default), then all packet sequences start at `0` for new connections. This reduces the cognitive load when debugging or troubleshooting the network stack.

Protocols like TCP randomize their starting sequence number by default. When `randomize_seq` is set to true, `westworld3` will also randomize the starting sequence number.

## connection_setup_timeout_ms

The `connection_setup_timeout_ms` parameter specifies the number of milliseconds within which a new connection setup attempt must complete. After the configured number of milliseconds, the connection is abandoned and an error is returned to the caller.

## connection_inactive_timeout_ms

The `connection_inactive_timeout_ms` parameter controls how long the `westworld3` protocol will wait after not having received any communication from its peer, before abandoning the connection and returning an error to the caller.

## send_keepalive

The `send_keepalive` switch determines whether or not keepalive packets are sent when an idle connection is detected. When keepalives are enabled, and the transmitter has not sent data for more than `(connection_inactive_timeout_ms / 2)`, a keepalive packet will be transmitted. Defaults to `true`.

## close_wait_ms

During the connection close process, the peer will wait `close_wait_ms` milliseconds before assuming the peer is gone, and abandoning the shutdown process, returning `EOF` to the caller.

## close_check_ms

`close_check_ms` determines the how frequently the shutdown state will be inspected by the connection close process. Defaults to `500ms`. In practice, this parameter should not require tuning.

## Transmitter Portal Mechanics

The `westworld3` "window" concept is referred to as a _portal_ ("portal" seems more appropriate in the "Transwarp" universe). 

The "portal size" refers to the amount of data that's allowed to be "in flight" within a link at any time. The transmitter considers the "available portal capacity" to be the current portal size, minus any data that's already in flight. Every time a new packet is transmitted, the available capacity is reduced by the size of the payload of that packet. When the transmitter receives an ACK from the receiver, the available capacity is increased by that amount.

The portal size is used to regulate the overall flow of data from the transmitter to the receiver.

## tx_portal_start_sz, tx_portal_min_sz, tx_portal_max_sz

The `tx_portal_start_sz`, `tx_portal_min_sz`, and `tx_portal_max_sz` parameters control the starting, minimum, and maximum sizes of the portal.

Setting the `tx_portal_start_sz` value too low will cause a longer "soft start" behavior after initial connection establishment. The portal will start small and will need to grow before the link can fill up to achieve high throughput.

Setting the `tx_portal_min_sz` value too low will cause a longer "soft start" after congestion control occurs. The protocol will "throttle back" more deeply and it will potentially take longer to achieve maximum throughput once nominal conditions return.

Setting the `tx_portal_max_sz` value too low will limit the upper end performance of a connection. Setting it too large can overrun the socket buffers, causing drastic decreases in overall performance.

Best guidance is to use the Westworld Analyzer and watch the `tx_portal_sz` to see how the portal size behaves in your real-world conditions.

## tx_portal_increase_thresh, tx_portal_increase_scale

In order for `westworld3` to transmit at a faster rate, the portal size needs to increase, allowing more packets to be in transit at any point in time. When the protocol is operating nominally, without congestion or loss signals being detected (retransmissions or duplicate ACKs), the stack will attempt to increase the size of the portal.

The transmitter counts the number of successfully ACKed packets. This counter is used to determine when to increase the portal size. The `tx_portal_increase_thresh` parameter is a threshold for the number of successfully ACKed packets that need to occur before the portal size is scaled.

`tx_portal_increase_scale` determines the scaling value of the portal when `tx_portal_increase_thresh` is reached. If `tx_portal_increase_scale` is set to `2`, then the portal will be scaled by a factor of 2 when `tx_portal_increase_thresh` successfully-ACKed packets are processed.

Too high of a `tx_portal_increase_scale` can cause the portal to scale up too quickly, faster than the protocol's ability to detect the link capacity exhaustion and can lead to sawtooth cycles of expansion and contraction.

It is best to balance how frequently the portal scales up (`tx_portal_increase_thresh`) and by how much (`tx_portal_increase_scale`) to tune the profile so that it saturates the link at a reasonable rate for the underlay type, and also in harmony with how the negative signalling scales the portal down in response to loss events.

## Congestion and Loss Signals

There are two primary signals used to determine when the link is experiencing congestion or loss. The first signal is a retransmission (`retx`) event. The only way for the transmitter to know when the receiver received a packet is when the transmitter receives an ACK message from the receiver. When the transmitter does not receive an ACK message from the transmitter within the expected time window, it will assume the receiver did not receive the packet and will retransmit it again. These retransmissions are counted.

The other signal used to determine when a link is experiencing congestion or loss is a duplicate ACK. We retransmit because we assume the receiver did not receive the packet. It's also possible that the receiver did receive the packet, but the ACK response was lost. And it's also possible that the receiver did receive the packet, but we did not get the ACK in the timeframe we expected. In the latter case, we'll end up getting a second ACK back in response to our retransmission. The receiver always ACKs a packet regardless if it has seen it before or not. Duplicate ACKs are also counted (separately).

The retransmission (`retx`) and duplicate ACK (`dupack`) counters are used for negative signalling to scale the portal size down.

## Duplicate ACK (`dupack`) Signal Scaling

The transmitter will scale the portal in response to duplicate ACK (`dupack`) events:

`tx_portal_dupack_thresh` is the number of duplicate ACK events that need to occur before the transmitter will scale the portal size down.

`tx_portal_dupack_capacity_scale` determines by _how much_ the portal will be scaled when `dupack_thresh` is reached. A value of `0.5` would reduce the portal size to half.

`tx_portal_dupack_success_scale` determines by _how much_ the success counter will be scaled when `dupack_thresh` is reached. If this is set to `0`, then the success counter (see `tx_portal_increase_thresh` above) will be set to `0` when `dupack_thresh` is reached.

Linking `tx_portal_increase_thresh` and `tx_portal_dupack_thresh` through `tx_portal_dupack_success_scale` allows duplicate ACK events to not only decrease the portal size, but also adjust how quickly the portal will scale back up when these events occur.

## Retransmission (`retx`) Signal Scaling

The transmitter will scale the portal in response to retransmission (`retx`) events:

The `tx_portal_retx_*` parameters control portal scaling in response to retransmission events in the same way as teh `tx_portal_dupack_*` parameters described above.

Differentiating between `retx` and `dupack` events allows the stack to be tuned to respond to those events differently.

## Tuning Portal Loss Signal Scaling

Both scaling the transmitter portal up when nominal, successful transmission occurs, and back down when loss or congestion is detected needs to be done iteratively and holistically through profiling and analysis. The `thresh` parameters control _how quickly_ the stack will react to the signals its detecting. Generally, it's probably most reasonable to scale up more slowly and scale down more quickly.

It's also important to balance the `scale` parameters against the `thresh` parameter. Using a lower threshold and a higher scale means the scaling will happen more frequently, but will change the portal size by a smaller amount.

There is a lot of operational practice to be built around how we use these parameters to directly control the posture of our overlay networks.

## Hysteresis

`westworld3` currently has only rudimentary ideas about hysteresis at this point. Future versions will likely endeavor to build a richer model of the perceived network performance and use observations over time to make more advanced assertions about the capabilities of an underlay link.

There's a lot we can do and learn about the performance of our overlay networks, without advanced self-tuning models.

## tx_portal_rx_sz_pressure_scale

The receiver maintains its own buffer size value. Payloads might arrive out of order at the receiver, which requires the receiver to buffer them until the "holes" in the buffer are filled by the missing packets. The current size of the receiver buffer is another secondary signal to the receiver that there are potentially issues with the underlay link. It also indicates back-pressure from the receiver's client (the application).

The receiver's buffer size is continually sent from the receiver to the transmitter by embedding it in the ACK messages sent by the receiver.

The `tx_portal_rx_sz_pressure_scale` value controls how much of the receiver's buffer size will be mixed into the calculation of the available portal capacity for the transmitter.

This is the calculation used to compute the available capacity for the transmitter:

available capacity = portal size - (receiver buffer size * `tx_portal_rx_sz_pressure_scale`) - size of un-ACKed payloads

## retx_start_ms

The retransmission interval is continually computed by sending round trip time probes across the link (embedded in the data stream and ACK responses). This ends up self-tuning the timeout between when an ACK is received for a packet, and when it is considered "lost" and a retransmission occurs.

When a connection is first established, there is no baseline RTT established. The `retx_start_ms` parameter controls the starting retransmission timeout. Until the RTT probes establish a usable baseline, retranmission interval will be `retx_start_ms`.

## Retransmission Deadline (`retx`) Scaling

`westworld3` continually tries to discover the tightest possible retransmission deadline by slowly scaling RTT probe. When a connection is running in a nominal state the protocol will slowly decrease the retransmission time scaling factor, trying to adjust the retransmission time to have the optimal responsiveness.

Retransmission that occurs too slowly will result in poor performance due to latency in receiving missing packets. Retransmission that occurs too eagerly will result in poor performance due to redundant data transmission. A number of factors including workloads, host environment, and changing network conditions can all shift the goal for the appropriate responsiveness. Retransmision scaling is an automaton that tries to continually seek the best retransmission time.

`retx_scale` defines the initial scaling factor for the retransmission time.

`retx_scale_floor` defines the minimum scaling factor for the retransmission time.

In the process of optimally scaling the `retx_scale_factor`, the protocol will periodically evaluate the positive and negative indicators (successful ACKs, `retx`, `dupack`) to determine if the `retx_scale` value should be increased or decreased.

`retx_evaluation_ms` configures how frequently these evaluations occur.

`retx_evaluation_scale_incr` determines how much the `retx_scale_factor` will be increased when duplicate ACKs are detected. Duplicate ACKs are indicative of too-eager retransmission.

`retx_evaluation_scale_decr` determines how much the `retx_scale_factor` will be decreased when nominal, effective transmission is occuring.

Both increases and decreases to the retransmission scaling factor are only applied no more frequently than every `retx_evaluation_ms`.

## retx_add_ms

`retx_add_ms` is a configured "overhead" for the retransmission time. This is a non-scaled, additive factor that's included to support a fixed amount of "processing overhead".

This value predates the `retx_scale` model. To use a fixed RTT time computation, the `retx_add_ms` value can be set to non-`0`, and the `retx_scale` values can all be set to stop the RTT scaling automaton from adjusting the computed `retx` deadline.

## retx_batch_ms

Packets to be retransmitted are held in an ordered queue by their deadline. When retranmission happens at the head of the queue, the retransmitter will continue to release retransmissions for events that are the current deadline, plus `retx_batch_ms` milliseconds.

So, if we're retransmitting a packet that's due for retransmission _right now_, and the next 50 messages are due _right now_ plus `2ms` and `retx_batch_ms` is set to `2ms`, then all of those packets will be retransmitted at once.

This is mostly an efficiency control for the retransmitter mechanism. A certain amount of timing slop happens when the retransmitter evaluates the next deadline. Using `retx_batch_ms` to release groups of retransmit events simultaneously can tighten retransmitter timing.

## rtt_probe_ms, rtt_probe_avg

The round-trip time (RTT) is probed periodically, and then averaged to compute the _retransmit time_. The `rtt_probe_ms` parameter controls how frequently the RTT is probed. The `rtt_probe_avg` is the number of RTT probes to average when computing the retransmit time.

## rx_portal_sz_pacing_thresh

When processing packets at the receiver, certain "out of order" conditions can free up large portions of the receiver's buffer (kind of like when you get that block just right in Tetris). When this happens, the transmitter can end up with a reduced portal and unable to transmit if the last ACK received indicated that the receiver's buffer was full (or nearly full).

If a payload receiption by the receiver causes the size of the receiver's buffer to change by more than `rx_portal_sz_pacing_thresh`, then it will automatically transmit a _pacing ACK_ (an empty ACK with just the receiver's buffer size) to allow the transmitter to continue transmitting.

## max_segment_sz

`max_segment_sz` controls how large the `westworld3` portion of the packet can be. `westworld3` is encoded in a UDP datagram, so the total size of the packet would include the underlay specific details. You'll want to ensure that you configure `max_segment_sz` small enough that your underlay overhead does not result in fragmented or dropped datagrams.

## Other Values

(`pool_buffer_sz`, `rx_buffer_sz`, `tx_buffer_sz`, `tx_portal_tree_len`, `retx_monitor_tree_len`, `rx_portal_tree_len`, `listener_peers_tree_len`, `reads_queue_len`, `listener_rx_queue_len`, `accept_queue_len`)

Internal configuration values. Should not require any tuning.