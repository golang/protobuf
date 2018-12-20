// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

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
	File           interface{}
	Enum           interface{}
	EnumValue      interface{}
	Message        interface{}
	Field          interface{}
	Oneof          interface{}
	ExtensionRange interface{}
	Service        interface{}
	Method         interface{}
}

func (internal) RegisterFileOptions(m interface{})           { optionTypes.File = m }
func (internal) RegisterEnumOptions(m interface{})           { optionTypes.Enum = m }
func (internal) RegisterEnumValueOptions(m interface{})      { optionTypes.EnumValue = m }
func (internal) RegisterMessageOptions(m interface{})        { optionTypes.Message = m }
func (internal) RegisterFieldOptions(m interface{})          { optionTypes.Field = m }
func (internal) RegisterOneofOptions(m interface{})          { optionTypes.Oneof = m }
func (internal) RegisterExtensionRangeOptions(m interface{}) { optionTypes.ExtensionRange = m }
func (internal) RegisterServiceOptions(m interface{})        { optionTypes.Service = m }
func (internal) RegisterMethodOptions(m interface{})         { optionTypes.Method = m }
