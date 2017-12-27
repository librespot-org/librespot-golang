// Code generated by protoc-gen-go. DO NOT EDIT.
// source: playlist4meta.proto

package Spotify

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type DownloadFormat_Codec int32

const (
	DownloadFormat_CODEC_UNKNOWN  DownloadFormat_Codec = 0
	DownloadFormat_OGG_VORBIS     DownloadFormat_Codec = 1
	DownloadFormat_FLAC           DownloadFormat_Codec = 2
	DownloadFormat_MPEG_1_LAYER_3 DownloadFormat_Codec = 3
)

var DownloadFormat_Codec_name = map[int32]string{
	0: "CODEC_UNKNOWN",
	1: "OGG_VORBIS",
	2: "FLAC",
	3: "MPEG_1_LAYER_3",
}
var DownloadFormat_Codec_value = map[string]int32{
	"CODEC_UNKNOWN":  0,
	"OGG_VORBIS":     1,
	"FLAC":           2,
	"MPEG_1_LAYER_3": 3,
}

func (x DownloadFormat_Codec) Enum() *DownloadFormat_Codec {
	p := new(DownloadFormat_Codec)
	*p = x
	return p
}
func (x DownloadFormat_Codec) String() string {
	return proto.EnumName(DownloadFormat_Codec_name, int32(x))
}
func (x *DownloadFormat_Codec) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(DownloadFormat_Codec_value, data, "DownloadFormat_Codec")
	if err != nil {
		return err
	}
	*x = DownloadFormat_Codec(value)
	return nil
}
func (DownloadFormat_Codec) EnumDescriptor() ([]byte, []int) { return fileDescriptor11, []int{1, 0} }

type ListChecksum struct {
	Version          *int32 `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Sha1             []byte `protobuf:"bytes,4,opt,name=sha1" json:"sha1,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *ListChecksum) Reset()                    { *m = ListChecksum{} }
func (m *ListChecksum) String() string            { return proto.CompactTextString(m) }
func (*ListChecksum) ProtoMessage()               {}
func (*ListChecksum) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{0} }

func (m *ListChecksum) GetVersion() int32 {
	if m != nil && m.Version != nil {
		return *m.Version
	}
	return 0
}

func (m *ListChecksum) GetSha1() []byte {
	if m != nil {
		return m.Sha1
	}
	return nil
}

type DownloadFormat struct {
	Codec            *DownloadFormat_Codec `protobuf:"varint,1,opt,name=codec,enum=Spotify.DownloadFormat_Codec" json:"codec,omitempty"`
	XXX_unrecognized []byte                `json:"-"`
}

func (m *DownloadFormat) Reset()                    { *m = DownloadFormat{} }
func (m *DownloadFormat) String() string            { return proto.CompactTextString(m) }
func (*DownloadFormat) ProtoMessage()               {}
func (*DownloadFormat) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{1} }

func (m *DownloadFormat) GetCodec() DownloadFormat_Codec {
	if m != nil && m.Codec != nil {
		return *m.Codec
	}
	return DownloadFormat_CODEC_UNKNOWN
}

type ListAttributes struct {
	Name                    *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Description             *string `protobuf:"bytes,2,opt,name=description" json:"description,omitempty"`
	Picture                 []byte  `protobuf:"bytes,3,opt,name=picture" json:"picture,omitempty"`
	Collaborative           *bool   `protobuf:"varint,4,opt,name=collaborative" json:"collaborative,omitempty"`
	Pl3Version              *string `protobuf:"bytes,5,opt,name=pl3_version,json=pl3Version" json:"pl3_version,omitempty"`
	DeletedByOwner          *bool   `protobuf:"varint,6,opt,name=deleted_by_owner,json=deletedByOwner" json:"deleted_by_owner,omitempty"`
	RestrictedCollaborative *bool   `protobuf:"varint,7,opt,name=restricted_collaborative,json=restrictedCollaborative" json:"restricted_collaborative,omitempty"`
	DeprecatedClientId      *int64  `protobuf:"varint,8,opt,name=deprecated_client_id,json=deprecatedClientId" json:"deprecated_client_id,omitempty"`
	PublicStarred           *bool   `protobuf:"varint,9,opt,name=public_starred,json=publicStarred" json:"public_starred,omitempty"`
	ClientId                *string `protobuf:"bytes,10,opt,name=client_id,json=clientId" json:"client_id,omitempty"`
	XXX_unrecognized        []byte  `json:"-"`
}

func (m *ListAttributes) Reset()                    { *m = ListAttributes{} }
func (m *ListAttributes) String() string            { return proto.CompactTextString(m) }
func (*ListAttributes) ProtoMessage()               {}
func (*ListAttributes) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{2} }

func (m *ListAttributes) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *ListAttributes) GetDescription() string {
	if m != nil && m.Description != nil {
		return *m.Description
	}
	return ""
}

func (m *ListAttributes) GetPicture() []byte {
	if m != nil {
		return m.Picture
	}
	return nil
}

func (m *ListAttributes) GetCollaborative() bool {
	if m != nil && m.Collaborative != nil {
		return *m.Collaborative
	}
	return false
}

func (m *ListAttributes) GetPl3Version() string {
	if m != nil && m.Pl3Version != nil {
		return *m.Pl3Version
	}
	return ""
}

func (m *ListAttributes) GetDeletedByOwner() bool {
	if m != nil && m.DeletedByOwner != nil {
		return *m.DeletedByOwner
	}
	return false
}

func (m *ListAttributes) GetRestrictedCollaborative() bool {
	if m != nil && m.RestrictedCollaborative != nil {
		return *m.RestrictedCollaborative
	}
	return false
}

func (m *ListAttributes) GetDeprecatedClientId() int64 {
	if m != nil && m.DeprecatedClientId != nil {
		return *m.DeprecatedClientId
	}
	return 0
}

func (m *ListAttributes) GetPublicStarred() bool {
	if m != nil && m.PublicStarred != nil {
		return *m.PublicStarred
	}
	return false
}

func (m *ListAttributes) GetClientId() string {
	if m != nil && m.ClientId != nil {
		return *m.ClientId
	}
	return ""
}

type ItemAttributes struct {
	AddedBy          *string         `protobuf:"bytes,1,opt,name=added_by,json=addedBy" json:"added_by,omitempty"`
	Timestamp        *int64          `protobuf:"varint,2,opt,name=timestamp" json:"timestamp,omitempty"`
	Message          *string         `protobuf:"bytes,3,opt,name=message" json:"message,omitempty"`
	Seen             *bool           `protobuf:"varint,4,opt,name=seen" json:"seen,omitempty"`
	DownloadCount    *int64          `protobuf:"varint,5,opt,name=download_count,json=downloadCount" json:"download_count,omitempty"`
	DownloadFormat   *DownloadFormat `protobuf:"bytes,6,opt,name=download_format,json=downloadFormat" json:"download_format,omitempty"`
	SevendigitalId   *string         `protobuf:"bytes,7,opt,name=sevendigital_id,json=sevendigitalId" json:"sevendigital_id,omitempty"`
	SevendigitalLeft *int64          `protobuf:"varint,8,opt,name=sevendigital_left,json=sevendigitalLeft" json:"sevendigital_left,omitempty"`
	SeenAt           *int64          `protobuf:"varint,9,opt,name=seen_at,json=seenAt" json:"seen_at,omitempty"`
	Public           *bool           `protobuf:"varint,10,opt,name=public" json:"public,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *ItemAttributes) Reset()                    { *m = ItemAttributes{} }
func (m *ItemAttributes) String() string            { return proto.CompactTextString(m) }
func (*ItemAttributes) ProtoMessage()               {}
func (*ItemAttributes) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{3} }

func (m *ItemAttributes) GetAddedBy() string {
	if m != nil && m.AddedBy != nil {
		return *m.AddedBy
	}
	return ""
}

func (m *ItemAttributes) GetTimestamp() int64 {
	if m != nil && m.Timestamp != nil {
		return *m.Timestamp
	}
	return 0
}

func (m *ItemAttributes) GetMessage() string {
	if m != nil && m.Message != nil {
		return *m.Message
	}
	return ""
}

func (m *ItemAttributes) GetSeen() bool {
	if m != nil && m.Seen != nil {
		return *m.Seen
	}
	return false
}

func (m *ItemAttributes) GetDownloadCount() int64 {
	if m != nil && m.DownloadCount != nil {
		return *m.DownloadCount
	}
	return 0
}

func (m *ItemAttributes) GetDownloadFormat() *DownloadFormat {
	if m != nil {
		return m.DownloadFormat
	}
	return nil
}

func (m *ItemAttributes) GetSevendigitalId() string {
	if m != nil && m.SevendigitalId != nil {
		return *m.SevendigitalId
	}
	return ""
}

func (m *ItemAttributes) GetSevendigitalLeft() int64 {
	if m != nil && m.SevendigitalLeft != nil {
		return *m.SevendigitalLeft
	}
	return 0
}

func (m *ItemAttributes) GetSeenAt() int64 {
	if m != nil && m.SeenAt != nil {
		return *m.SeenAt
	}
	return 0
}

func (m *ItemAttributes) GetPublic() bool {
	if m != nil && m.Public != nil {
		return *m.Public
	}
	return false
}

type StringAttribute struct {
	Key              *string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value            *string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *StringAttribute) Reset()                    { *m = StringAttribute{} }
func (m *StringAttribute) String() string            { return proto.CompactTextString(m) }
func (*StringAttribute) ProtoMessage()               {}
func (*StringAttribute) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{4} }

func (m *StringAttribute) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *StringAttribute) GetValue() string {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return ""
}

type StringAttributes struct {
	Attribute        []*StringAttribute `protobuf:"bytes,1,rep,name=attribute" json:"attribute,omitempty"`
	XXX_unrecognized []byte             `json:"-"`
}

func (m *StringAttributes) Reset()                    { *m = StringAttributes{} }
func (m *StringAttributes) String() string            { return proto.CompactTextString(m) }
func (*StringAttributes) ProtoMessage()               {}
func (*StringAttributes) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{5} }

func (m *StringAttributes) GetAttribute() []*StringAttribute {
	if m != nil {
		return m.Attribute
	}
	return nil
}

func init() {
	proto.RegisterType((*ListChecksum)(nil), "Spotify.ListChecksum")
	proto.RegisterType((*DownloadFormat)(nil), "Spotify.DownloadFormat")
	proto.RegisterType((*ListAttributes)(nil), "Spotify.ListAttributes")
	proto.RegisterType((*ItemAttributes)(nil), "Spotify.ItemAttributes")
	proto.RegisterType((*StringAttribute)(nil), "Spotify.StringAttribute")
	proto.RegisterType((*StringAttributes)(nil), "Spotify.StringAttributes")
	proto.RegisterEnum("Spotify.DownloadFormat_Codec", DownloadFormat_Codec_name, DownloadFormat_Codec_value)
}

func init() { proto.RegisterFile("playlist4meta.proto", fileDescriptor11) }

var fileDescriptor11 = []byte{
	// 640 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x52, 0x5d, 0x6f, 0x13, 0x3b,
	0x10, 0xbd, 0xe9, 0x36, 0x4d, 0x32, 0x69, 0xb7, 0x5b, 0xdf, 0xea, 0x76, 0xaf, 0x00, 0x11, 0x45,
	0x20, 0x22, 0x21, 0x45, 0xb4, 0x41, 0x48, 0x95, 0x78, 0x20, 0xdd, 0x7e, 0x10, 0x08, 0x0d, 0x72,
	0x44, 0x11, 0x4f, 0x2b, 0x67, 0x3d, 0x69, 0xad, 0xee, 0x97, 0x6c, 0x27, 0x55, 0x5e, 0xf9, 0x03,
	0xfc, 0x4c, 0xfe, 0x06, 0xb2, 0xb3, 0xf9, 0xaa, 0xc4, 0x9b, 0xcf, 0x99, 0x33, 0xe3, 0x99, 0x33,
	0x03, 0xff, 0xe6, 0x31, 0x9b, 0xc5, 0x42, 0xe9, 0xb7, 0x09, 0x6a, 0xd6, 0xce, 0x65, 0xa6, 0x33,
	0x52, 0x19, 0xe6, 0x99, 0x16, 0xe3, 0x59, 0xf3, 0x3d, 0xec, 0xf6, 0x85, 0xd2, 0xc1, 0x1d, 0x46,
	0xf7, 0x6a, 0x92, 0x10, 0x1f, 0x2a, 0x53, 0x94, 0x4a, 0x64, 0xa9, 0x5f, 0x6a, 0x94, 0x5a, 0x65,
	0xba, 0x80, 0x84, 0xc0, 0xb6, 0xba, 0x63, 0xc7, 0xfe, 0x76, 0xa3, 0xd4, 0xda, 0xa5, 0xf6, 0xdd,
	0xfc, 0x55, 0x02, 0xf7, 0x3c, 0x7b, 0x48, 0xe3, 0x8c, 0xf1, 0xcb, 0x4c, 0x26, 0x4c, 0x93, 0x0e,
	0x94, 0xa3, 0x8c, 0x63, 0x64, 0xd3, 0xdd, 0x93, 0x67, 0xed, 0xe2, 0xa7, 0xf6, 0xa6, 0xae, 0x1d,
	0x18, 0x11, 0x9d, 0x6b, 0x9b, 0x1f, 0xa1, 0x6c, 0x31, 0x39, 0x80, 0xbd, 0x60, 0x70, 0x7e, 0x11,
	0x84, 0xdf, 0xae, 0x3f, 0x5f, 0x0f, 0xbe, 0x5f, 0x7b, 0xff, 0x10, 0x17, 0x60, 0x70, 0x75, 0x15,
	0xde, 0x0c, 0xe8, 0x59, 0x6f, 0xe8, 0x95, 0x48, 0x15, 0xb6, 0x2f, 0xfb, 0xdd, 0xc0, 0xdb, 0x22,
	0x04, 0xdc, 0x2f, 0x5f, 0x2f, 0xae, 0xc2, 0xe3, 0xb0, 0xdf, 0xfd, 0x71, 0x41, 0xc3, 0x8e, 0xe7,
	0x34, 0x7f, 0x3a, 0xe0, 0x9a, 0x81, 0xba, 0x5a, 0x4b, 0x31, 0x9a, 0x68, 0x54, 0xa6, 0xf1, 0x94,
	0x25, 0x68, 0x1b, 0xaa, 0x51, 0xfb, 0x26, 0x0d, 0xa8, 0x73, 0x54, 0x91, 0x14, 0xb9, 0x36, 0xa3,
	0x6e, 0xd9, 0xd0, 0x3a, 0x65, 0x8c, 0xc8, 0x45, 0xa4, 0x27, 0x12, 0x7d, 0xc7, 0x4e, 0xbc, 0x80,
	0xe4, 0x05, 0xec, 0x45, 0x59, 0x1c, 0xb3, 0x51, 0x26, 0x99, 0x16, 0x53, 0xb4, 0x8e, 0x54, 0xe9,
	0x26, 0x49, 0x9e, 0x43, 0x3d, 0x8f, 0x3b, 0xe1, 0xc2, 0xcc, 0xb2, 0xfd, 0x01, 0xf2, 0xb8, 0x73,
	0x53, 0xf8, 0xd9, 0x02, 0x8f, 0x63, 0x8c, 0x1a, 0x79, 0x38, 0x9a, 0x85, 0xd9, 0x43, 0x8a, 0xd2,
	0xdf, 0xb1, 0x95, 0xdc, 0x82, 0x3f, 0x9b, 0x0d, 0x0c, 0x4b, 0x4e, 0xc1, 0x97, 0xa8, 0xb4, 0x14,
	0x91, 0x11, 0x6f, 0xfe, 0x5d, 0xb1, 0x19, 0x47, 0xab, 0x78, 0xb0, 0xd1, 0xc5, 0x1b, 0x38, 0xe4,
	0x98, 0x4b, 0x8c, 0x98, 0x4d, 0x8d, 0x05, 0xa6, 0x3a, 0x14, 0xdc, 0xaf, 0x36, 0x4a, 0x2d, 0x87,
	0x92, 0x55, 0x2c, 0xb0, 0xa1, 0x1e, 0x27, 0x2f, 0xc1, 0xcd, 0x27, 0xa3, 0x58, 0x44, 0xa1, 0xd2,
	0x4c, 0x4a, 0xe4, 0x7e, 0x6d, 0x3e, 0xde, 0x9c, 0x1d, 0xce, 0x49, 0xf2, 0x04, 0x6a, 0xab, 0x6a,
	0x60, 0x87, 0xab, 0x46, 0x45, 0x8d, 0xe6, 0xef, 0x2d, 0x70, 0x7b, 0x1a, 0x93, 0xb5, 0x25, 0xfc,
	0x0f, 0x55, 0xc6, 0xb9, 0x9d, 0xb5, 0x58, 0x44, 0xc5, 0xe2, 0xb3, 0x19, 0x79, 0x0a, 0x35, 0x2d,
	0x12, 0x54, 0x9a, 0x25, 0xb9, 0xdd, 0x84, 0x43, 0x57, 0x84, 0xd9, 0x43, 0x82, 0x4a, 0xb1, 0xdb,
	0xf9, 0x1e, 0x6a, 0x74, 0x01, 0xed, 0x41, 0x22, 0xa6, 0x85, 0xfd, 0xf6, 0x6d, 0xba, 0xe7, 0xc5,
	0x9d, 0x85, 0x51, 0x36, 0x49, 0xb5, 0x35, 0xde, 0xa1, 0x7b, 0x0b, 0x36, 0x30, 0x24, 0xf9, 0x00,
	0xfb, 0x4b, 0xd9, 0xd8, 0xde, 0xa3, 0xb5, 0xbe, 0x7e, 0x72, 0xf4, 0x97, 0x73, 0xa5, 0xcb, 0xb2,
	0xc5, 0x99, 0xbf, 0x82, 0x7d, 0x85, 0x53, 0x4c, 0xb9, 0xb8, 0x15, 0x9a, 0xc5, 0xc6, 0x85, 0x8a,
	0x6d, 0xcf, 0x5d, 0xa7, 0x7b, 0x9c, 0xbc, 0x86, 0x83, 0x0d, 0x61, 0x8c, 0x63, 0x5d, 0xd8, 0xef,
	0xad, 0x07, 0xfa, 0x38, 0xd6, 0xe4, 0x08, 0x2a, 0x66, 0x8c, 0x90, 0x69, 0xeb, 0xba, 0x43, 0x77,
	0x0c, 0xec, 0x6a, 0xf2, 0x1f, 0xec, 0xcc, 0xfd, 0xb7, 0x5e, 0x57, 0x69, 0x81, 0x9a, 0xa7, 0xb0,
	0x3f, 0xd4, 0x52, 0xa4, 0xb7, 0x4b, 0xab, 0x89, 0x07, 0xce, 0x3d, 0x2e, 0x4c, 0x36, 0x4f, 0x72,
	0x08, 0xe5, 0x29, 0x8b, 0x27, 0x58, 0x9c, 0xf9, 0x1c, 0x34, 0x3f, 0x81, 0xf7, 0x28, 0x55, 0x91,
	0x77, 0x50, 0x63, 0x0b, 0xe4, 0x97, 0x1a, 0x4e, 0xab, 0x7e, 0xe2, 0x2f, 0x1d, 0x79, 0xa4, 0xa6,
	0x2b, 0xe9, 0x9f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x46, 0xce, 0xac, 0x0e, 0x64, 0x04, 0x00, 0x00,
}