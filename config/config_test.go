package config

import (
	"testing"
)

func toString(c *Configuration) string {
	if c == nil {
		return ""
	}
	var str string
	for oKey, origCollection := range c.Config {
		str += oKey
		for _, val := range origCollection {
			str += val.ContentType + val.Collection
		}
	}
	return str
}

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name     string
		confPath string
		wantC    *Configuration
		wantErr  bool
	}{
		{
			"Test1",
			"config.json",
			&Configuration{
				Config: map[string][]OriginSystemConfig{
					"http://cmdb.ft.com/systems/methode-web-pub": []OriginSystemConfig{
						{ContentType: ".*",
							Collection: "methode",
						},
					},
				}},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := ReadConfig(tt.confPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if toString(gotC) != toString(tt.wantC) {
				t.Errorf("ReadConfig() = %v, want %v", gotC, tt.wantC)
			}
		})
	}
}
