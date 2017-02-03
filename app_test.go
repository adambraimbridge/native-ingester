package main

import (
	"encoding/json"
	"testing"
)

func TestExtractUuid_NormalCase(t *testing.T) {
	var tests = []struct {
		msgBody string
		paths   []string
		uuid    string
	}{
		{
			"{" +
				"\"uuid\": \"07ac9fad-6434-47c7-b7c4-34361a048d07\"" +
				"}",
			[]string{"uuid"},
			"07ac9fad-6434-47c7-b7c4-34361a048d07",
		},
		{
			"{" +
				"  \"post\": {" +
				"    \"uuid\": \"07ac9fad-6434-47c7-b7c4-34361a048d07\"" +
				"  }" +
				"}",
			[]string{"uuid", "post.uuid"},
			"07ac9fad-6434-47c7-b7c4-34361a048d07",
		},
		{
			"{" +
				"  \"data\": {" +
				"    \"uuidv3\": \"07ac9fad-6434-47c7-b7c4-34361a048d07\"" +
				"  }" +
				"}",
			[]string{"uuid", "post.uuid", "data.uuidv3"},
			"07ac9fad-6434-47c7-b7c4-34361a048d07",
		},
		{
			"{" +
				"  \"data\": {" +
				"    \"name\": \"John\"" +
				"  }" +
				"}",
			[]string{"uuid", "post.uuid", "data.uuidv3"},
			"",
		},
	}
	for _, test := range tests {
		contents := make(map[string]interface{})
		json.Unmarshal([]byte(test.msgBody), &contents)
		actualUUID := extractUUID(contents, test.paths)
		if actualUUID != test.uuid {
			t.Errorf("Error extracting uuid.\nHeader: %s\nExpected: %s\nActual: %s\n", test.msgBody, test.uuid, actualUUID)
		}
	}
}
