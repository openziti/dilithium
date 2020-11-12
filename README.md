# The Dilithium Project

_The Dilithium Project_ is a software framework and a set of tooling that supports the development of high-performance streaming protocols over unreliable message-passing systems (UDP datagrams for example).

Initially, we're focusing on the development of `westworld`, which is a high-performance streaming protocol designed to maximize throughput across non-ideal wide-area network links.

Ultimately, _Dilithium_ will become an extensible framework (implemented in `golang`) for the development of high-performance communication over any arbitrary unreliable message passing system. 

_Dilithium_ feeds into the [OpenZiti](https://github.com/openziti) project, in support of it's performance-oriented _Transwarp_ data plane implementation. In this role, _Dilithium_ provides both the UDP-based implementation, and also manages the higher-level overlay abstractions implemented as messages on top of existing streaming protocols.

## Concepts

For a more detailed tour of _Dilithium_ and the framework components it provides, see the [Concepts Guide](docs/concepts.md).

![Loss Handling](docs/images/concepts/loss_handling.png)