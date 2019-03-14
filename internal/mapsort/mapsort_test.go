// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mapsort_test

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/v2/internal/mapsort"
	"github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func TestRange(t *testing.T) {
	for _, test := range []struct {
		mapv interface{}
		kind pref.Kind
	}{
		{
			mapv: &map[bool]int32{
				false: 0,
				true:  1,
			},
			kind: pref.BoolKind,
		},
		{
			mapv: &map[int32]int32{
				0: 0,
				1: 1,
				2: 2,
			},
			kind: pref.Int32Kind,
		},
		{
			mapv: &map[uint64]int32{
				0: 0,
				1: 1,
				2: 2,
			},
			kind: pref.Uint64Kind,
		},
		{
			mapv: &map[string]int32{
				"a": 0,
				"b": 1,
				"c": 2,
			},
			kind: pref.StringKind,
		},
	} {
		rv := reflect.TypeOf(test.mapv).Elem()
		mapv := value.MapOf(test.mapv, value.NewConverter(rv.Key(), test.kind), value.NewConverter(rv.Elem(), pref.Int32Kind))
		var got []pref.MapKey
		mapsort.Range(mapv, test.kind, func(key pref.MapKey, _ pref.Value) bool {
			got = append(got, key)
			return true
		})
		for i, key := range got {
			if int64(i) != mapv.Get(key).Int() {
				t.Errorf("out of order range over map: %v", got)
				break
			}
		}
	}
}
