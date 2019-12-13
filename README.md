# Simplified and enhanced Go support for Protocol Buffer v3 encoding

Derived from the canonical google github.com/golang/protobuf, with
the encoder and decoders rewritten to:

- Support arrays, not just slices
- Support struct fields, not just pointers-to-struct 
- Support custom marshalers better by encoding the key, and only needing the value
  to be supplied by the custom marshaler. And support slices of marshalers, as well
  as slices that can marshal themselves.
- Support ignored (unmarshaled) fields
- Support embedded struct fields
- Only support protobuf v3 (simplier, faster marshaling and unmarshaling)
- Generate .proto files from the go struct definitions.
- Error checking to support hand edited `protobuf:"..."` field tags by folks who
  don't know protobuf very well.

and whatever else I have found useful to implement along the way. For example:

    var s = struct {
      X int    `protobuf:"varint,1"`
      Y string `protobuf:"bytes,2"`
    }{
      X: 7,
      Y: "hello",
    }
    
    pb,err := protobuf3.Marshal(&s)

encodes to 0807120568656c6c6f, the protobuf representation of s.



The unit tests in github.com/mistsys/protobuf3/protobuf3/unit_test.go exercise
all the functionality, and serve as examples.

The code you want is in github.com/mistsys/protobuf3/protobuf3 package. The
stutter is historical.

This code started as a fork of github.com/golang/protobuf because I wanted
to encode deeply nested data which had heretofore been marshaled into JSON.
The JSON marshalers, it turns out, permit many more of the data structures
which Go has, and which we used, than golang/protobuf could handle.

At this point very little of the original is left, and this package can
be used stand-alone.

Orignal code before the fork is Copyright 2010 The Go Authors.

Modifications are Copyright 2016-2019 Mist Systems.

