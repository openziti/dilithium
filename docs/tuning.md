# Westworld3 Tuning Guide

This guide explains some of the `westworld3` profile parameters and how to think about their impact on the performance of the protocol.

This is what a complete `westworld3.Profile` definition looks like:

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

If the `randomize_seq` switch is set to `false` (the default), then all packet sequences start at `0` for new connections. This makes debugging and troubleshooting the stack simpler.

Protocols like TCP randomize the starting sequence number by default. Turning `randomize_seq` on by setting the value to `true`, will randomize the starting sequence number.

## connection_setup_timeout_ms

The `connection_setup_timeout_ms` parameter specifies the number of milliseconds within which a new connection setup must complete. After the configured number of milliseconds, the connection is abandoned and an error is returned to the caller.

## connection_inactive_timeout_ms

The `connection_inactive_timeout_ms` parameter controls how long the `westworld3` protocol will wait after not having received any communication from its peer, before abandoning the connection and returning an error to the caller.

## send_keepalive

The `send_keepalive` switch determines whether or not keepalives are sent when an idle connection is detected. When keepalives are enabled, and the transmitter has not sent data for more than `connection_inactive_timeout_ms / 2`, a keepalive will be transmitted. Defaults to `true`.

## close_wait_ms

During the connection close process, the peer will wait `close_wait_ms` milliseconds before abandoning the shutdown process and returning `EOF` to the caller.

## close_check_ms

`close_check_ms` determines how frequently the close process will inspect the timeout state of the close process. Defaults to `500ms`. Should not require tuning, unless the application requires very responsive close processing timeouts.

## tx_portal_start_sz, tx_portal_min_sz, tx_portal_max_sz

The `westworld3` "window" concept is referred to as a _portal_. The `tx_portal_start_sz`, `tx_portal_min_sz`, and `tx_portal_max_sz` control the starting, minimum, and maximum sizes of the portal.

Setting `tx_portal_start_sz` too low will cause more "soft start" behavior at connection setup. The portal will start small and will need to grow before the link can fill up to achieve high throughput.

Setting `tx_portal_max_sz` too low will limit the upper end performance of a connection. Setting it too large can overrun the socket buffers, causing drastic decreases in overall performance.

Best guidance is to use the Westworld Analyzer and watch the `tx_portal_sz` to see how the portal size behaves.

## tx_portal_increase_thresh, tx_portal_increase_scale

The transmitter will increase the portal size when it processes `tx_portal_increase_thresh` successfully ACKed packets, with no retransmissions or duplicate ACKs. Retransmissions and duplicate ACKs reset the success counter to `0`.

`tx_portal_increase_scale` determines the scaling value of the portal when `tx_portal_increase_thresh` is reached. If `tx_portal_increase_scale` is set to `2`, then the portal will be scaled by a factor of 2 when `tx_portal_increase_thresh` successfully-ACKed packets are processed.

Too high of a `tx_portal_increase_scale` can cause the portal to scale up too quickly, faster than the protocol's ability to detect the link capacity exhaustion and can lead to cycles of congestion and contraction.

It is best to balance how frequently the portal scales up (`tx_portal_increase_thresh`) and by how much (`tx_portal_increase_scale`) to tune the profile so that it saturates the link at a reasonable rate for the underlay type.

## tx_portal_dupack_thresh, tx_portal_dupack_capacity_scale, tx_portal_dupack_success_scale

`tx_portal_dupack_thresh` is the number of duplicate ACK events that need to occur before the transmitter will scale the portal size down.

`tx_portal_dupack_capacity_scale` determines by _how much_ the portal will be scaled when `dupack_thresh` is reached. A value of `0.5` would reduce the portal size to half.

`tx_portal_dupack_success_scale` determines by _how much_ the success counter will be scaled when `dupack_thresh` is reached. If this is set to `0`, then the success counter (see `tx_portal_increase_thresh` above) will be set to `0` when `dupack_thresh` is reached.

Linking `tx_portal_increase_thresh` and `tx_portal_dupack_thresh` through `tx_portal_dupack_success_scale` allows duplicate ACK events to not only decrease the portal size, but also adjust how quickly the portal will scale back up when these events occur.

## tx_portal_retx_thresh, tx_portal_retx_capacity_scale, tx_portal_retx_success_scale

See `tx_portal_dupack_*` above. These values work exactly the same, but instead of duplicate ACKs working as the trigger, retransmission events are used.

## tx_portal_rx_sz_pressure_scale

`tx_portal_rx_sz_pressure_scale` allows some proportion of the receiver's portal size to be mixed into the calculation of the available link capacity for the transmitter.

This has the effect of allowing the transmitter to respond more quickly to local congestion at the receiver, and also to cases where out-of-order delivery are impacting the receiver.

The receiver portal size is continually delivered to the transmitter through ACK responses.

## retx_start_ms

The retransmission interval is continually computed by sending round trip time probes across the link (embedded in the data stream and ACK responses). This ends up self-tuning the timeout between when an ACK is received for a packet, and when it is considered "lost" and a retransmission occurs.

When a connection is first established, there is no baseline RTT. The `retx_start_ms` parameter controls the starting retransmission timeout. Until the RTT probes establish a functional baseline, the baseline will be set to this value.

## retx_scale, retx_scale_floor

`westworld3` continually tries to discover the appropriate RTT "slop factor" by slowly scaling the retransmission timeout. When a connection is running in a nominal state the protocol will slowly decrease the retransmission time scaling factor, trying to adjust the retransmission time to have the optimal responsiveness.

Retransmission that occurs too slowly will result in poor performance. Retransmission that occurs too eagerly will result in poor performance. A number of factors including workloads, host environment, and changing network conditions can all shift the goal for the appropriate responsiveness. Retransmision scaling is an automaton that tries to continually seek the best retransmission time.

`retx_scale` defines the initial scaling factor for the retransmission time.

`retx_scale_floor` defines the minimum scaling factor for the retransmission time.

## retx_add_ms

`retx_add_ms` is a configured "overhead" for the retransmission time. This is a non-scaled, additive factor that's included to support a fixed amount of "processing overhead".

## retx_evaluation_ms, retx_evaluation_scale_incr, retx_evaluation_scale_decr

In the process of optimally scaling the `retx_scale_factor`, the protocol will periodically evaluate.

`retx_evaluation_ms` configures how frequently these evaluations occur.

`retx_evaluation_scale_incr` determines how much the `retx_scale_factor` will be increased when duplicate ACKs are detected. Duplicate ACKs are indicative of too-eager retransmission.

`retx_evaluation_scale_decr` determines how much the `retx_scale_factor` will be decreased when nominal, effective transmission is occuring.

Both increases and decreases to the retransmission scaling factor are only applied no more frequently than every `retx_evaluation_ms`.

## retx_batch_ms

Packets to be retransmitted are held in an ordered queue by their deadline. When retranmission happens at the head of the queue, the retransmitter will continue to release retransmissions for events that are the current deadline, plus `retx_batch_ms` milliseconds.

So, if we're retransmitting a packet that's due for retransmission _right now_, and the next 50 messages are due _right now_ plus `2ms` and `retx_batch_ms` is set to `2ms`, then all of those packets will be retransmitted at once.

This is mostly an efficiency control for the retransmitter mechanism. A certain amount of timing slop happens when the retransmitter evaluates the next deadline. Using `retx_batch_ms` to release groups of retransmit events simultaneously can tighten retransmitter timing.

## rtt_probe_ms, rtt_probe_avg

The round-trip time (RTT) is probed periodically, and then averaged to compute the _retransmit time_. The `rtt_probe_ms` parameter controls how frequently the RTT is probed. The `rtt_probe_avg` is the number of RTT probes to average when computing the retransmit time.

## rx_portal_sz_pacing_thresh

`rx_portal_sz_pacing_thresh` controls how frequently we encode the receiver portal size into the ACK messages returning to the transmitter. Specified in milliseconds.

It's not free to encode a 4-byte number into the returning ACK messages, so we don't want to encode it on every ACK. But encoding the receiver portal size too infrequently into the returning ACKs means that the transmitter has delayed visibility into congestion issues at the receiver. Encoding the receiver portal size too frequently consumes available bandwidth.

## max_segment_sz

`max_segment_sz` controls how large the `westworld3` portion of the packet can be. `westworld3` is encoded in a UDP datagram, so the total size of the packet would include the underlay specific details. You'll want to ensure that you configure `max_segment_sz` small enough that your underlay overhead does not result in fragmented or dropped datagrams.

## pool_buffer_sz,rx_buffer_sz, tx_buffer_sz, tx_portal_tree_len, retx_monitor_tree_len, rx_portal_tree_len, listener_peers_tree_len, reads_queue_len, listener_rx_queue_len, accept_queue_len

Internal configuration values. Should not require any tuning.