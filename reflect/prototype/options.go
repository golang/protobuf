// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import pref "github.com/golang/protobuf/v2/reflect/protoreflect"

// X provides functionality internal to the protobuf module.
//
// WARNING: The compatibility agreement covers nothing except for functionality
// needed to keep existing generated messages operational. The Go authors
// are not responsible for breakages that occur due to unauthorized usages.
var X internal

type internal struct{}

// optionTypes contains typed nil-pointers to each of the options types.
// These are populated at init time by the descriptor package.
var optionTypes struct {
	File           pref.ProtoMessage
	Enum           pref.ProtoMessage
	EnumValue      pref.ProtoMessage
	Message        pref.ProtoMessage
	Field          pref.ProtoMessage
	Oneof          pref.ProtoMessage
	ExtensionRange pref.ProtoMessage
	Service        pref.ProtoMessage
	Method         pref.ProtoMessage
}

func (internal) RegisterFileOptions(m pref.ProtoMessage)           { optionTypes.File = m }
func (internal) RegisterEnumOptions(m pref.ProtoMessage)           { optionTypes.Enum = m }
func (internal) RegisterEnumValueOptions(m pref.ProtoMessage)      { optionTypes.EnumValue = m }
func (internal) RegisterMessageOptions(m pref.ProtoMessage)        { optionTypes.Message = m }
func (internal) RegisterFieldOptions(m pref.ProtoMessage)          { optionTypes.Field = m }
func (internal) RegisterOneofOptions(m pref.ProtoMessage)          { optionTypes.Oneof = m }
func (internal) RegisterExtensionRangeOptions(m pref.ProtoMessage) { optionTypes.ExtensionRange = m }
func (internal) RegisterServiceOptions(m pref.ProtoMessage)        { optionTypes.Service = m }
func (internal) RegisterMethodOptions(m pref.ProtoMessage)         { optionTypes.Method = m }
