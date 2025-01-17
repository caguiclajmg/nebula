// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.7
// source: cert.proto

package cert

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RawNebulaCertificate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Details   *RawNebulaCertificateDetails `protobuf:"bytes,1,opt,name=Details,proto3" json:"Details,omitempty"`
	Signature []byte                       `protobuf:"bytes,2,opt,name=Signature,proto3" json:"Signature,omitempty"`
}

func (x *RawNebulaCertificate) Reset() {
	*x = RawNebulaCertificate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cert_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RawNebulaCertificate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RawNebulaCertificate) ProtoMessage() {}

func (x *RawNebulaCertificate) ProtoReflect() protoreflect.Message {
	mi := &file_cert_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RawNebulaCertificate.ProtoReflect.Descriptor instead.
func (*RawNebulaCertificate) Descriptor() ([]byte, []int) {
	return file_cert_proto_rawDescGZIP(), []int{0}
}

func (x *RawNebulaCertificate) GetDetails() *RawNebulaCertificateDetails {
	if x != nil {
		return x.Details
	}
	return nil
}

func (x *RawNebulaCertificate) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type RawNebulaCertificateDetails struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Names []string `protobuf:"bytes,1,rep,name=Names,proto3" json:"Names,omitempty"`
	// Ips and Subnets are in big endian 32 bit pairs, 1st the ip, 2nd the mask
	Ips       []uint32 `protobuf:"varint,2,rep,packed,name=Ips,proto3" json:"Ips,omitempty"`
	Subnets   []uint32 `protobuf:"varint,3,rep,packed,name=Subnets,proto3" json:"Subnets,omitempty"`
	Groups    []string `protobuf:"bytes,4,rep,name=Groups,proto3" json:"Groups,omitempty"`
	NotBefore int64    `protobuf:"varint,5,opt,name=NotBefore,proto3" json:"NotBefore,omitempty"`
	NotAfter  int64    `protobuf:"varint,6,opt,name=NotAfter,proto3" json:"NotAfter,omitempty"`
	PublicKey []byte   `protobuf:"bytes,7,opt,name=PublicKey,proto3" json:"PublicKey,omitempty"`
	IsCA      bool     `protobuf:"varint,8,opt,name=IsCA,proto3" json:"IsCA,omitempty"`
	// sha-256 of the issuer certificate, if this field is blank the cert is self-signed
	Issuer []byte `protobuf:"bytes,9,opt,name=Issuer,proto3" json:"Issuer,omitempty"`
}

func (x *RawNebulaCertificateDetails) Reset() {
	*x = RawNebulaCertificateDetails{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cert_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RawNebulaCertificateDetails) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RawNebulaCertificateDetails) ProtoMessage() {}

func (x *RawNebulaCertificateDetails) ProtoReflect() protoreflect.Message {
	mi := &file_cert_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RawNebulaCertificateDetails.ProtoReflect.Descriptor instead.
func (*RawNebulaCertificateDetails) Descriptor() ([]byte, []int) {
	return file_cert_proto_rawDescGZIP(), []int{1}
}

func (x *RawNebulaCertificateDetails) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

func (x *RawNebulaCertificateDetails) GetIps() []uint32 {
	if x != nil {
		return x.Ips
	}
	return nil
}

func (x *RawNebulaCertificateDetails) GetSubnets() []uint32 {
	if x != nil {
		return x.Subnets
	}
	return nil
}

func (x *RawNebulaCertificateDetails) GetGroups() []string {
	if x != nil {
		return x.Groups
	}
	return nil
}

func (x *RawNebulaCertificateDetails) GetNotBefore() int64 {
	if x != nil {
		return x.NotBefore
	}
	return 0
}

func (x *RawNebulaCertificateDetails) GetNotAfter() int64 {
	if x != nil {
		return x.NotAfter
	}
	return 0
}

func (x *RawNebulaCertificateDetails) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *RawNebulaCertificateDetails) GetIsCA() bool {
	if x != nil {
		return x.IsCA
	}
	return false
}

func (x *RawNebulaCertificateDetails) GetIssuer() []byte {
	if x != nil {
		return x.Issuer
	}
	return nil
}

var File_cert_proto protoreflect.FileDescriptor

var file_cert_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x63, 0x65, 0x72, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x63, 0x65,
	0x72, 0x74, 0x22, 0x71, 0x0a, 0x14, 0x52, 0x61, 0x77, 0x4e, 0x65, 0x62, 0x75, 0x6c, 0x61, 0x43,
	0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x12, 0x3b, 0x0a, 0x07, 0x44, 0x65,
	0x74, 0x61, 0x69, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x63, 0x65,
	0x72, 0x74, 0x2e, 0x52, 0x61, 0x77, 0x4e, 0x65, 0x62, 0x75, 0x6c, 0x61, 0x43, 0x65, 0x72, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x44, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x52, 0x07,
	0x44, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x53, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x53, 0x69, 0x67, 0x6e,
	0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0xfb, 0x01, 0x0a, 0x1b, 0x52, 0x61, 0x77, 0x4e, 0x65, 0x62,
	0x75, 0x6c, 0x61, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x44, 0x65,
	0x74, 0x61, 0x69, 0x6c, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x49,
	0x70, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x03, 0x49, 0x70, 0x73, 0x12, 0x18, 0x0a,
	0x07, 0x53, 0x75, 0x62, 0x6e, 0x65, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x07,
	0x53, 0x75, 0x62, 0x6e, 0x65, 0x74, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x47, 0x72, 0x6f, 0x75, 0x70,
	0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x12,
	0x1c, 0x0a, 0x09, 0x4e, 0x6f, 0x74, 0x42, 0x65, 0x66, 0x6f, 0x72, 0x65, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x09, 0x4e, 0x6f, 0x74, 0x42, 0x65, 0x66, 0x6f, 0x72, 0x65, 0x12, 0x1a, 0x0a,
	0x08, 0x4e, 0x6f, 0x74, 0x41, 0x66, 0x74, 0x65, 0x72, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x08, 0x4e, 0x6f, 0x74, 0x41, 0x66, 0x74, 0x65, 0x72, 0x12, 0x1c, 0x0a, 0x09, 0x50, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x50, 0x75,
	0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x49, 0x73, 0x43, 0x41, 0x18,
	0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x49, 0x73, 0x43, 0x41, 0x12, 0x16, 0x0a, 0x06, 0x49,
	0x73, 0x73, 0x75, 0x65, 0x72, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x49, 0x73, 0x73,
	0x75, 0x65, 0x72, 0x42, 0x20, 0x5a, 0x1e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x73, 0x6c, 0x61, 0x63, 0x6b, 0x68, 0x71, 0x2f, 0x6e, 0x65, 0x62, 0x75, 0x6c, 0x61,
	0x2f, 0x63, 0x65, 0x72, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cert_proto_rawDescOnce sync.Once
	file_cert_proto_rawDescData = file_cert_proto_rawDesc
)

func file_cert_proto_rawDescGZIP() []byte {
	file_cert_proto_rawDescOnce.Do(func() {
		file_cert_proto_rawDescData = protoimpl.X.CompressGZIP(file_cert_proto_rawDescData)
	})
	return file_cert_proto_rawDescData
}

var file_cert_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_cert_proto_goTypes = []interface{}{
	(*RawNebulaCertificate)(nil),        // 0: cert.RawNebulaCertificate
	(*RawNebulaCertificateDetails)(nil), // 1: cert.RawNebulaCertificateDetails
}
var file_cert_proto_depIdxs = []int32{
	1, // 0: cert.RawNebulaCertificate.Details:type_name -> cert.RawNebulaCertificateDetails
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_cert_proto_init() }
func file_cert_proto_init() {
	if File_cert_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cert_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RawNebulaCertificate); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cert_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RawNebulaCertificateDetails); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cert_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cert_proto_goTypes,
		DependencyIndexes: file_cert_proto_depIdxs,
		MessageInfos:      file_cert_proto_msgTypes,
	}.Build()
	File_cert_proto = out.File
	file_cert_proto_rawDesc = nil
	file_cert_proto_goTypes = nil
	file_cert_proto_depIdxs = nil
}
