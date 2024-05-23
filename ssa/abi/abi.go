/*
 * Copyright (c) 2024 The GoPlus Authors (goplus.org). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package abi

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"go/types"
	"hash"
)

// -----------------------------------------------------------------------------

type Kind int

const (
	Invalid  Kind = iota
	Indirect      // allocate memory for the value
	Pointer       // store a pointer value directly in the interface value
	Integer       // store a integer value directly in the interface value
	BitCast       // store other value (need bitcast) directly in the interface value
)

func KindOf(raw types.Type, lvl int, is32Bits bool) (Kind, types.Type, int) {
	switch t := raw.Underlying().(type) {
	case *types.Basic:
		kind := t.Kind()
		switch {
		case types.Bool <= kind && kind <= types.Uintptr:
			if is32Bits && (kind == types.Int64 || kind == types.Uint64) {
				return Indirect, raw, lvl
			}
			return Integer, raw, lvl
		case kind == types.Float32:
			return BitCast, raw, lvl
		case kind == types.Float64 || kind == types.Complex64:
			if is32Bits {
				return Indirect, raw, lvl
			}
			return BitCast, raw, lvl
		case kind == types.UnsafePointer:
			return Pointer, raw, lvl
		}
	case *types.Pointer, *types.Signature, *types.Map, *types.Chan:
		return Pointer, raw, lvl
	case *types.Struct:
		if t.NumFields() == 1 {
			return KindOf(t.Field(0).Type(), lvl+1, is32Bits)
		}
	case *types.Interface, *types.Slice:
	case *types.Array:
		if t.Len() == 1 {
			return KindOf(t.Elem(), lvl+1, is32Bits)
		}
	default:
		panic("unkown type")
	}
	return Indirect, raw, lvl
}

// -----------------------------------------------------------------------------

// Builder is a helper for constructing ABI types.
type Builder struct {
	h   hash.Hash
	buf []byte
	Pkg string
}

// New creates a new ABI type Builder.
func New(pkg string) *Builder {
	ret := new(Builder)
	ret.Init(pkg)
	return ret
}

func (b *Builder) Init(pkg string) {
	b.Pkg = pkg
	b.h = sha256.New()
	b.buf = make([]byte, sha256.Size)
}

// TypeName returns the ABI type name for the specified type.
func (b *Builder) TypeName(t types.Type) (ret string, private bool) {
	switch t := t.(type) {
	case *types.Basic:
		return BasicName(t), false
	case *types.Pointer:
		ret, private = b.TypeName(t.Elem())
		return "*" + ret, private
	case *types.Struct:
		return b.StructName(t)
	}
	panic("todo")
}

func BasicName(t *types.Basic) string {
	return "_llgo_" + t.Name()
}

// StructName returns the ABI type name for the specified struct type.
func (b *Builder) StructName(t *types.Struct) (ret string, private bool) {
	hash, private := b.structHash(t)
	hashStr := base64.RawURLEncoding.EncodeToString(hash)
	if private {
		return b.Pkg + ".struct$" + hashStr, true
	}
	return "_llgo_struct$" + hashStr, false
}

func (b *Builder) structHash(t *types.Struct) (ret []byte, private bool) {
	h := b.h
	h.Reset()
	n := t.NumFields()
	fmt.Fprintln(h, "struct", n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if !f.Exported() {
			private = true
		}
		name := f.Name()
		if f.Embedded() {
			name = "-"
		}
		ft, fpriv := b.TypeName(f.Type())
		if fpriv {
			private = true
		}
		fmt.Fprintln(h, name, ft)
	}
	ret = h.Sum(b.buf[:0])
	return
}

// -----------------------------------------------------------------------------
