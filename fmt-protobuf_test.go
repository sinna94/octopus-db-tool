package main

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"testing"
)

var protobufTplTestSchema = &Schema{
	Tables: []*Table{
		{
			Name: "user",
			Columns: []*Column{
				{
					Name:            "id",
					Type:            "long",
					PrimaryKey:      true,
					AutoIncremental: true,
				},
				{
					Name:      "name",
					Type:      "string",
					Size:      100,
					UniqueKey: true,
				},
				{
					Name:  "dec",
					Type:  "decimal",
					Size:  20,
					Scale: 5,
				},
				{
					Name: "created_at",
					Type: "datetime",
				},
				{
					Name:     "updated_at",
					Type:     "datetime",
					Nullable: true,
				},
			},
			Description: "",
			Group:       "common",
		},
	},
}

// data class
func TestProtobufTpl_Generate(t *testing.T) {
	output := &Output{
		Options: map[string]string{
			FlagPackage:   "com.lechuck.foo",
			FlagGoPackage: "lechuck/foo",
		},
	}
	prefixMapper := newPrefixMapper("common:C")
	expected := []string{
		`syntax = "proto3";

package com.lechuck.hello;

option go_package = "proto/hello";

import "google/protobuf/timestamp.proto";

message CUser {
  int64 id = 1;
  string name = 2;
  double dec = 3;
  google.protobuf.Timestamp createdAt = 4;
  google.protobuf.Timestamp updatedAt = 5;
}
`,
	}

	protobuf := NewProtobufTpl(protobufTplTestSchema, output, nil, prefixMapper)

	for i, table := range jpaKotlinTplTestSchema.Tables {
		messages := []*ProtoMessage{
			NewProtobufMessage(table, output, prefixMapper),
		}
		buf := new(bytes.Buffer)
		if err := protobuf.GenerateProto(buf, messages, "com.lechuck.hello", "proto/hello"); err != nil {
			t.Error(err)
		}
		actual := buf.String()
		if diff := cmp.Diff(expected[i], actual); diff != "" {
			t.Errorf("mismatch [%d] (-expected +actual):\n%s", i, diff)
		}
	}
}