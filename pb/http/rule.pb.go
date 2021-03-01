// Code generated by protoc-gen-go-ext. DO NOT EDIT.
// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: pb/http/rule.proto

package http

import (
	proto "github.com/golang/protobuf/proto"
	router "github.com/infraboard/mcube/http/router"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

var file_pb_http_rule_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.MethodOptions)(nil),
		ExtensionType: (*router.Entry)(nil),
		Field:         20210228,
		Name:          "mcube.http.rest_api",
		Tag:           "bytes,20210228,opt,name=rest_api",
		Filename:      "pb/http/rule.proto",
	},
}

// Extension fields to descriptorpb.MethodOptions.
var (
	// optional mcube.router.Entry rest_api = 20210228;
	E_RestApi = &file_pb_http_rule_proto_extTypes[0]
)

var File_pb_http_rule_proto protoreflect.FileDescriptor

var file_pb_http_rule_proto_rawDesc = []byte{
	0x0a, 0x12, 0x70, 0x62, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x72, 0x75, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6d, 0x63, 0x75, 0x62, 0x65, 0x2e, 0x68, 0x74, 0x74, 0x70,
	0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x15, 0x70, 0x62, 0x2f, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x2f, 0x65, 0x6e,
	0x74, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x3a, 0x51, 0x0a, 0x08, 0x72, 0x65, 0x73,
	0x74, 0x5f, 0x61, 0x70, 0x69, 0x12, 0x1e, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xb4, 0xc4, 0xd1, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13,
	0x2e, 0x6d, 0x63, 0x75, 0x62, 0x65, 0x2e, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x2e, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x07, 0x72, 0x65, 0x73, 0x74, 0x41, 0x70, 0x69, 0x42, 0x73, 0x0a, 0x13,
	0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x42, 0x0a, 0x48, 0x74, 0x74, 0x70, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x48,
	0x01, 0x5a, 0x28, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x69, 0x6e,
	0x66, 0x72, 0x61, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2f, 0x6d, 0x63, 0x75, 0x62, 0x65, 0x2f, 0x70,
	0x62, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x3b, 0x68, 0x74, 0x74, 0x70, 0xf8, 0x01, 0x01, 0xa2, 0x02,
	0x03, 0x47, 0x50, 0x42, 0xaa, 0x02, 0x1a, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x52, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_pb_http_rule_proto_goTypes = []interface{}{
	(*descriptorpb.MethodOptions)(nil), // 0: google.protobuf.MethodOptions
	(*router.Entry)(nil),               // 1: mcube.router.Entry
}
var file_pb_http_rule_proto_depIdxs = []int32{
	0, // 0: mcube.http.rest_api:extendee -> google.protobuf.MethodOptions
	1, // 1: mcube.http.rest_api:type_name -> mcube.router.Entry
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	1, // [1:2] is the sub-list for extension type_name
	0, // [0:1] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_pb_http_rule_proto_init() }
func file_pb_http_rule_proto_init() {
	if File_pb_http_rule_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pb_http_rule_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_pb_http_rule_proto_goTypes,
		DependencyIndexes: file_pb_http_rule_proto_depIdxs,
		ExtensionInfos:    file_pb_http_rule_proto_extTypes,
	}.Build()
	File_pb_http_rule_proto = out.File
	file_pb_http_rule_proto_rawDesc = nil
	file_pb_http_rule_proto_goTypes = nil
	file_pb_http_rule_proto_depIdxs = nil
}
