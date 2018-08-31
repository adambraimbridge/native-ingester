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
					"http://cmdb.ft.com/systems/next-video-editor": []OriginSystemConfig{
						{ContentType: "application/json",
							Collection: "video",
						},
						{ContentType: "^(application/)*(vnd.ft-upp-audio\\+json).*$",
							Collection: "audio",
						},
					},
				},
			},
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

func TestConfiguration_GetCollection(t *testing.T) {
	type args struct {
		originID    string
		contentType string
	}
	c := &Configuration{
		Config: map[string][]OriginSystemConfig{
			"http://cmdb.ft.com/systems/methode-web-pub": []OriginSystemConfig{
				{ContentType: ".*",
					Collection: "methode",
				},
			},
			"http://cmdb.ft.com/systems/next-video-editor": []OriginSystemConfig{
				//{ContentType: "^(application/json).*$",
				{ContentType: "application/json",
					Collection: "video",
				},
				{ContentType: "^(application/)*(vnd.ft-upp-audio\\+json).*$",
					Collection: "audio",
				},
			},
		},
	}
	err := c.validateConfig()
	if err != nil {
		t.Errorf("Configuration.GetCollection() error = %v", err)
		return
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"methode json",
			args{"http://cmdb.ft.com/systems/methode-web-pub",
				"application/json"},
			"methode",
			false,
			// TODO: Add test cases.
		},
		{
			"methode null CT",
			args{"http://cmdb.ft.com/systems/methode-web-pub",
				""},
			"methode",
			false,
			// TODO: Add test cases.
		},
		{
			"methode",
			args{"http://cmdb.ft.com/systems/methode-web-pub",
				"anytype"},
			"methode",
			false,
			// TODO: Add test cases.
		},
		{
			"video wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"anytype"},
			"",
			true,
			// TODO: Add test cases.
		},
		{
			"video OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json"},
			"video",
			false,
			// TODO: Add test cases.
		},
		{
			"video OK long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json; utf8"},
			"video",
			false,
			// TODO: Add test cases.
		},
		{
			"audio OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json"},
			"audio",
			false,
			// TODO: Add test cases.
		},
		{
			"audio long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json;UTF8"},
			"audio",
			false,
			// TODO: Add test cases.
		},
		{
			"audio wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio-json"},
			"",
			true,
			// TODO: Add test cases.
		},
		{
			"wrong origin",
			args{"http://cmdb.ft.com/systems/next",
				"application/vnd.ft-upp-audio+json"},
			"",
			true,
			// TODO: Add test cases.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := c.GetCollection(tt.args.originID, tt.args.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.GetCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Configuration.GetCollection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigurationMetadata_GetCollection(t *testing.T) {
	type args struct {
		originID    string
		contentType string
	}
	c := &Configuration{
		Config: map[string][]OriginSystemConfig{
			"http://cmdb.ft.com/systems/methode-web-pub": []OriginSystemConfig{
				{ContentType: ".*",
					Collection: "v1-metadata",
				},
			},
			"http://cmdb.ft.com/systems/next-video-editor": []OriginSystemConfig{
				{ContentType: "application/json",
					Collection: "video-metadata",
				},
			},
		},
	}
	err := c.validateConfig()
	if err != nil {
		t.Errorf("Configuration.GetCollection() error = %v", err)
		return
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"methode json",
			args{"http://cmdb.ft.com/systems/methode-web-pub",
				"application/json"},
			"v1-metadata",
			false,
			// TODO: Add test cases.
		},
		{
			"methode null CT",
			args{"http://cmdb.ft.com/systems/methode-web-pub",
				""},
			"v1-metadata",
			false,
			// TODO: Add test cases.
		},
		{
			"methode",
			args{"http://cmdb.ft.com/systems/methode-web-pub",
				"anytype"},
			"v1-metadata",
			false,
			// TODO: Add test cases.
		},
		{
			"video wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"anytype"},
			"",
			true,
			// TODO: Add test cases.
		},
		{
			"video OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json"},
			"video-metadata",
			false,
			// TODO: Add test cases.
		},
		{
			"video OK long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json; utf8"},
			"video-metadata",
			false,
			// TODO: Add test cases.
		},
		{
			"audio OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json"},
			"",
			true,
			// TODO: Add test cases.
		},
		{
			"audio long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json;UTF8"},
			"",
			true,
			// TODO: Add test cases.
		},
		{
			"audio wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio-json"},
			"",
			true,
			// TODO: Add test cases.
		},
		{
			"wrong origin",
			args{"http://cmdb.ft.com/systems/next",
				"application/vnd.ft-upp-audio+json"},
			"",
			true,
			// TODO: Add test cases.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := c.GetCollection(tt.args.originID, tt.args.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.GetCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Configuration.GetCollection() = %v, want %v", got, tt.want)
			}
		})
	}
}
