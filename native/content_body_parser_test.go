package native

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var happyTests = []struct {
	msgBody      string
	paths        []string
	expectedUUID string
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
}

var unhappyTests = []struct {
	msgBody string
	paths   []string
}{
	{
		"{" +
			"  \"data\": {" +
			"    \"name\": \"John\"" +
			"  }" +
			"}",
		[]string{"uuid", "post.uuid", "data.uuidv3"},
	},
	{
		"{" +
			"  \"data\": {" +
			"    \"name\": \"John\"" +
			"  }" +
			"}",
		[]string{},
	},
	{
		"{" +
			"  \"data\": {" +
			"    \"uuidv3\": \"07ac9fad-6434-47c7-b7c4-34361a048d07\"" +
			"  }" +
			"}",
		[]string{"uuid", "post.uuid"},
	},
}

func TestExtractUUIDSuccessfully(t *testing.T) {
	for _, test := range happyTests {
		bodyParser := NewContentBodyParser(test.paths)
		body := ContentBody{}
		json.Unmarshal([]byte(test.msgBody), &body)
		actualUUID, err := bodyParser.getUUID(body)
		assert.NoError(t, err, "The parsing should not return an error")
		assert.Equal(t, test.expectedUUID, actualUUID, "The UUIDs should be the same")
	}
}

func TestExtractUUIDFailure(t *testing.T) {
	for _, test := range unhappyTests {
		bodyParser := NewContentBodyParser(test.paths)
		body := ContentBody{}
		json.Unmarshal([]byte(test.msgBody), &body)
		_, err := bodyParser.getUUID(body)
		assert.Error(t, err, "The parsing should return an error")
	}
}
