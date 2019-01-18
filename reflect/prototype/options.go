// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

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
	File           pref.OptionsMessage
	Enum           pref.OptionsMessage
	EnumValue      pref.OptionsMessage
	Message        pref.OptionsMessage
	Field          pref.OptionsMessage
	Oneof          pref.OptionsMessage
	ExtensionRange pref.OptionsMessage
	Service        pref.OptionsMessage
	Method         pref.OptionsMessage
}

func (internal) FileOptions() pref.OptionsMessage           { return optionTypes.File }
func (internal) EnumOptions() pref.OptionsMessage           { return optionTypes.Enum }
func (internal) EnumValueOptions() pref.OptionsMessage      { return optionTypes.EnumValue }
func (internal) MessageOptions() pref.OptionsMessage        { return optionTypes.Message }
func (internal) FieldOptions() pref.OptionsMessage          { return optionTypes.Field }
func (internal) OneofOptions() pref.OptionsMessage          { return optionTypes.Oneof }
func (internal) ExtensionRangeOptions() pref.OptionsMessage { return optionTypes.ExtensionRange }
func (internal) ServiceOptions() pref.OptionsMessage        { return optionTypes.Service }
func (internal) MethodOptions() pref.OptionsMessage         { return optionTypes.Method }

func (internal) RegisterFileOptions(m pref.OptionsMessage)           { optionTypes.File = m }
func (internal) RegisterEnumOptions(m pref.OptionsMessage)           { optionTypes.Enum = m }
func (internal) RegisterEnumValueOptions(m pref.OptionsMessage)      { optionTypes.EnumValue = m }
func (internal) RegisterMessageOptions(m pref.OptionsMessage)        { optionTypes.Message = m }
func (internal) RegisterFieldOptions(m pref.OptionsMessage)          { optionTypes.Field = m }
func (internal) RegisterOneofOptions(m pref.OptionsMessage)          { optionTypes.Oneof = m }
func (internal) RegisterExtensionRangeOptions(m pref.OptionsMessage) { optionTypes.ExtensionRange = m }
func (internal) RegisterServiceOptions(m pref.OptionsMessage)        { optionTypes.Service = m }
func (internal) RegisterMethodOptions(m pref.OptionsMessage)         { optionTypes.Method = m }
