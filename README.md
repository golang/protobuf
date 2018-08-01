# Next Generation Go Protocol Buffers

**WARNING**: This repository is in active development. There are no guarantees
about API stability. Breaking changes *will* occur until a stable release is
made and announced.

This repository is for the development of the next major Go implementation
of protocol buffers. This library makes breaking API changes relative to the
[existing Go protobuf library](https://github.com/golang/protobuf/tree/master).
Of particular note, this API aims to make protobuf reflection a first-class
feature of the API and implements the protobuf ecosystem in terms of reflection.

# Design Documents

List of relevant design documents:

* [Go Protocol Buffer API Improvements](https://goo.gl/wwiPVx)
* [Go Reflection API for Protocol Buffers](https://goo.gl/96gGnE)

# Contributing

We appreciate community contributions. See [CONTRIBUTING.md](CONTRIBUTING.md).

# Reporting Issues

Issues regarding the new API can be filed at
[github.com/golang/protobuf](https://github.com/golang/protobuf/issues).
Please use a `APIv2:` prefix in the title to make it clear that
the issue is regarding the new API work.
