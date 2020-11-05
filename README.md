# The Dilithium Project

_The Dilithium Project_ is a set of tooling that supports the development of high-performance streaming protocols over unreliable message-passing systems (UDP datagrams, for example).

Initially, we're focusing on the development of `westworld2` and `westworld3`, which are high-performance streaming protocols designed to maximize throughput across wide-area network links.

Ultimately, _Dilithium_ will become an extensible framework (implemented in `golang`) for the development of high-performance streaming over any arbitrary unreliable message passing system. 

_Dilithium_ feeds into the [OpenZiti](https://github.com/openziti) project, in support of it's high-performance _Transwarp_ data plane implementation. In this role, _Dilithium_ provides both the high-performance UDP-based implementation, and also manages the higher-level overlay abstractions implemented as messages on top of existing streaming protocols.

## Concepts

Here's a picture of some things. This is a work im progress. For a more detailed tour of _Dilithium_ and the framework components it provides, see the [Concepts Guide](docs/concepts.md).

![Loss Handling](docs/images/concepts/loss_handling.png)