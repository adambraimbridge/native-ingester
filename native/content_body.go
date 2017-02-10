package native

// ContentBody represents the body of a native piece of content
type ContentBody map[string]interface{}

func (body ContentBody) publishReference() string {
	return body["lastModified"].(string)
}
