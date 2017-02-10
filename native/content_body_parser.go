package native

import (
	"errors"
	"strings"

	"github.com/jmoiron/jsonq"
)

//ContentBodyParser parses the body of native content
type ContentBodyParser interface {
	getUUID(body map[string]interface{}) (string, error)
}

type contentBodyParser struct {
	uuidJSONPaths []string
}

//NewContentBodyParser returns a new instace of a ContentBodyParser
func NewContentBodyParser(uuidJSONPaths []string) ContentBodyParser {
	return &contentBodyParser{uuidJSONPaths}
}

func (p contentBodyParser) getUUID(body map[string]interface{}) (string, error) {
	jq := jsonq.NewQuery(body)
	for _, uuidPath := range p.uuidJSONPaths {
		uuid, err := jq.String(strings.Split(uuidPath, ".")...)
		if err == nil && uuid != "" {
			return uuid, nil
		}
	}
	return "", errors.New("UUID not found")
}
