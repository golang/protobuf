# Simplified and enhanced Go support for Protocol Buffer v3 encoding

Derived from the canonical google github.com/golang/protobuf, with
the encoder and decoders rewritten to:

- Support arrays, not just slices
- Support struct fields, not just pointers-to-struct 
- Support custom marshalers better by encoding the key and only needed the value
  from the custom marshaler. And support slices of marshalers, as well as slices
  that can marshal themselves.
- Support ignored (not marshaled) fields
- Support embedded struct fields
- Only support protobuf v3
- Generate .proto files from the go struct definitions.
- Error checking to support hand edited `protobuf:"..."` field tags by folks who
  don't know protobuf very well.

and whatever else I may find useful along the way.

The code you want is in github.com/mistsys/protobuf3/protobuf3 package.

This code started as a fork of github.com/golang/protobuf because I wanted
to encode deeply nested data which had heretofore been marshaled into JSON.
The JSON marshalers, it turns out, permit many more of the data structures
which Go has, and which we used, than golang/protobuf could handle.

At this point very little of the original is left, and this package can
be used stand-alone.

Orignal code before the fork is Copyright 2010 The Go Authors.
Modifications are Copyright 2016-2019 Mist Systems.

