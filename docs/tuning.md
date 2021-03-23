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

