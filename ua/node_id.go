// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// todo(fs): fix mask

// NodeID is an identifier for a node in the address space of an OPC UA Server.
// The NodeID object encodes all different node id types.
type NodeID struct {
	Mask NodeIDType
	Ns   uint16
	Nid  uint32
	Bid  []byte
	Gid  *GUID
}

// NewTwoByteNodeID returns a new two byte node id.
func NewTwoByteNodeID(id uint8) *NodeID {
	return &NodeID{
		Mask: NodeIDTypeTwoByte,
		Nid:  uint32(id),
	}
}

// NewFourByteNodeID returns a new four byte node id.
func NewFourByteNodeID(ns uint8, id uint16) *NodeID {
	return &NodeID{
		Mask: NodeIDTypeFourByte,
		Ns:   uint16(ns),
		Nid:  uint32(id),
	}
}

// NewNumericNodeID returns a new numeric node id.
func NewNumericNodeID(ns uint16, id uint32) *NodeID {
	return &NodeID{
		Mask: NodeIDTypeNumeric,
		Ns:   ns,
		Nid:  id,
	}
}

// NewStringNodeID returns a new string node id.
func NewStringNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		Mask: NodeIDTypeString,
		Ns:   ns,
		Bid:  []byte(id),
	}
}

// NewGUIDNodeID returns a new GUID node id.
func NewGUIDNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		Mask: NodeIDTypeGUID,
		Ns:   ns,
		Gid:  NewGUID(id),
	}
}

// NewByteStringNodeID returns a new byte string node id.
func NewByteStringNodeID(ns uint16, id []byte) *NodeID {
	return &NodeID{
		Mask: NodeIDTypeByteString,
		Ns:   ns,
		Bid:  id,
	}
}

// ParseNodeID returns a node id from a string definition of the format
// 'ns=<namespace>;{s,i,b,g}=<identifier>'.
//
// For string node ids the 's=' prefix can be omitted.
//
// For numeric ids the smallest possible type which can store the namespace
// and id value is returned.
//
// Namespace URLs 'nsu=' are not supported since they require a lookup.
//
func ParseNodeID(s string) (*NodeID, error) {
	if s == "" {
		return NewTwoByteNodeID(0), nil
	}

	p := strings.SplitN(s, ";", 2)
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid node id: %s", s)
	}
	nsval, idval := p[0], p[1]

	// parse namespace
	var ns uint16
	switch {
	case strings.HasPrefix(nsval, "nsu="):
		return nil, fmt.Errorf("namespace urls are not supported: %s", s)

	case strings.HasPrefix(nsval, "ns="):
		n, err := strconv.Atoi(nsval[3:])
		if err != nil {
			return nil, fmt.Errorf("invalid namespace id: %s", s)
		}
		if n < 0 || n > math.MaxUint16 {
			return nil, fmt.Errorf("namespace id out of range (0..65535): %s", s)
		}
		ns = uint16(n)

	default:
		return nil, fmt.Errorf("invalid node id: %s", s)
	}

	// parse identifier
	switch {
	case strings.HasPrefix(idval, "i="):
		id, err := strconv.ParseUint(idval[2:], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric id: %s", s)
		}
		switch {
		case ns == 0 && id < 256:
			return NewTwoByteNodeID(byte(id)), nil
		case ns < 256 && id < math.MaxUint16:
			return NewFourByteNodeID(byte(ns), uint16(id)), nil
		case id < math.MaxUint32:
			return NewNumericNodeID(ns, uint32(id)), nil
		default:
			return nil, fmt.Errorf("numeric id out of range (0..2^32-1): %s", s)
		}

	case strings.HasPrefix(idval, "s="):
		return NewStringNodeID(ns, idval[2:]), nil

	case strings.HasPrefix(idval, "g="):
		n := NewGUIDNodeID(ns, idval[2:])
		if n == nil || n.StringID() == "" {
			return nil, fmt.Errorf("invalid guid node id: %s", s)
		}
		return n, nil

	case strings.HasPrefix(idval, "b="):
		b, err := base64.StdEncoding.DecodeString(idval[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid opaque node id: %s", s)
		}
		return NewByteStringNodeID(ns, b), nil

	default:
		return NewStringNodeID(ns, idval), nil
	}
}

// EncodingMask returns the encoding mask field including the
// type information and additional flags.
func (n *NodeID) EncodingMask() NodeIDType {
	return n.Mask
}

// Type returns the node id type in EncodingMask.
func (n *NodeID) Type() NodeIDType {
	return n.Mask & NodeIDType(0xf)
}

// URIFlag returns whether the URI flag is set in EncodingMask.
func (n *NodeID) URIFlag() bool {
	return n.Mask&0x80 == 0x80
}

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (n *NodeID) SetURIFlag() {
	n.Mask |= 0x80
}

// IndexFlag returns whether the Index flag is set in EncodingMask.
func (n *NodeID) IndexFlag() bool {
	return n.Mask&0x40 == 0x40
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (n *NodeID) SetIndexFlag() {
	n.Mask |= 0x40
}

// Namespace returns the namespace id. For two byte node ids
// this will always be zero.
func (n *NodeID) Namespace() uint16 {
	return n.Ns
}

// SetNamespace sets the namespace id. It returns an error
// if the id is not within the range of the node id type.
func (n *NodeID) SetNamespace(v uint16) error {
	switch n.Type() {
	case NodeIDTypeTwoByte:
		if v != 0 {
			return fmt.Errorf("out of range [0..0]: %d", v)
		}
		return nil

	case NodeIDTypeFourByte:
		if max := uint16(math.MaxUint8); v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.Ns = uint16(v)
		return nil

	default:
		if max := uint16(math.MaxUint16); v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.Ns = uint16(v)
		return nil
	}
}

// IntID returns the identifier value if the type is
// TwoByte, FourByte or Numeric. For all other types IntID
// returns 0.
func (n *NodeID) IntID() uint32 {
	return n.Nid
}

// SetIntID sets the identifier value for two byte, four byte and
// numeric node ids. It returns an error for other types.
func (n *NodeID) SetIntID(v uint32) error {
	switch n.Type() {
	case NodeIDTypeTwoByte:
		if max := uint32(math.MaxUint8); v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.Nid = uint32(v)
		return nil

	case NodeIDTypeFourByte:
		if max := uint32(math.MaxUint16); v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.Nid = uint32(v)
		return nil

	case NodeIDTypeNumeric:
		if max := uint32(math.MaxUint32); v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.Nid = uint32(v)
		return nil

	default:
		return fmt.Errorf("incompatible node id type")
	}
}

// StringID returns the string value of the identifier
// for String and GUID NodeIDs, and the base64 encoded
// value for Opaque types. For all other types StringID
// returns an empty string.
func (n *NodeID) StringID() string {
	switch n.Type() {
	case NodeIDTypeGUID:
		if n.Gid == nil {
			return ""
		}
		return n.Gid.String()
	case NodeIDTypeString:
		return string(n.Bid)
	case NodeIDTypeByteString:
		return base64.StdEncoding.EncodeToString(n.Bid)
	default:
		return ""
	}
}

// SetStringID sets the identifier value for string, guid and opaque
// node ids. It returns an error for other types.
func (n *NodeID) SetStringID(v string) error {
	switch n.Type() {
	case NodeIDTypeGUID:
		n.Gid = NewGUID(v)
		return nil

	case NodeIDTypeString:
		n.Bid = []byte(v)
		return nil

	case NodeIDTypeByteString:
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}
		n.Bid = b
		return nil

	default:
		return fmt.Errorf("incompatible node id type")
	}
}

// String returns the string representation of the NodeID
// in the format described by ParseNodeID.
func (n *NodeID) String() string {
	switch n.Type() {
	case NodeIDTypeTwoByte:
		return fmt.Sprintf("i=%d", n.Nid)

	case NodeIDTypeFourByte:
		if n.Ns == 0 {
			return fmt.Sprintf("i=%d", n.Nid)
		}
		return fmt.Sprintf("ns=%d;i=%d", n.Ns, n.Nid)

	case NodeIDTypeNumeric:
		if n.Ns == 0 {
			return fmt.Sprintf("i=%d", n.Nid)
		}
		return fmt.Sprintf("ns=%d;i=%d", n.Ns, n.Nid)

	case NodeIDTypeString:
		if n.Ns == 0 {
			return fmt.Sprintf("s=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;s=%s", n.Ns, n.StringID())

	case NodeIDTypeGUID:
		if n.Ns == 0 {
			return fmt.Sprintf("g=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;g=%s", n.Ns, n.StringID())

	case NodeIDTypeByteString:
		if n.Ns == 0 {
			return fmt.Sprintf("o=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;o=%s", n.Ns, n.StringID())

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

func (n *NodeID) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)

	n.Mask = NodeIDType(buf.ReadByte())
	typ := n.Mask & 0xf

	switch typ {
	case NodeIDTypeTwoByte:
		n.Nid = uint32(buf.ReadByte())
		return buf.Pos(), buf.Error()

	case NodeIDTypeFourByte:
		n.Ns = uint16(buf.ReadByte())
		n.Nid = uint32(buf.ReadUint16())
		return buf.Pos(), buf.Error()

	case NodeIDTypeNumeric:
		n.Ns = buf.ReadUint16()
		n.Nid = buf.ReadUint32()
		return buf.Pos(), buf.Error()

	case NodeIDTypeGUID:
		n.Ns = buf.ReadUint16()
		n.Gid = &GUID{}
		buf.ReadStruct(n.Gid)
		return buf.Pos(), buf.Error()

	case NodeIDTypeByteString, NodeIDTypeString:
		n.Ns = buf.ReadUint16()
		n.Bid = buf.ReadBytes()
		return buf.Pos(), buf.Error()

	default:
		return 0, fmt.Errorf("invalid node id type %v", typ)
	}
}

func (n *NodeID) Encode() ([]byte, error) {
	buf := NewBuffer(nil)
	buf.WriteByte(byte(n.Mask))

	switch n.Type() {
	case NodeIDTypeTwoByte:
		buf.WriteByte(byte(n.Nid))
	case NodeIDTypeFourByte:
		buf.WriteByte(byte(n.Ns))
		buf.WriteUint16(uint16(n.Nid))
	case NodeIDTypeNumeric:
		buf.WriteUint16(n.Ns)
		buf.WriteUint32(n.Nid)
	case NodeIDTypeGUID:
		buf.WriteUint16(n.Ns)
		buf.WriteStruct(n.Gid)
	case NodeIDTypeByteString, NodeIDTypeString:
		buf.WriteUint16(n.Ns)
		buf.WriteByteString(n.Bid)
	default:
		return nil, fmt.Errorf("invalid node id type %v", n.Type())
	}
	return buf.Bytes(), buf.Error()
}
