package main

import (
	"encoding/json"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const testBodyRawMsgValue = `{"uuid":"e7a3b814-59ee-459e-8f60-517f3e80ed99","type":"Image","value":"wkKt5ATGuDJYGfvTc28VUjs9g7SD+l8fqcqdgCup2uGfbMyfExoHpNf25Vd9QMo0JVgjp/YhQl+1PgPfcLhpJG41T+K2PL3JpXFwV0TfALn0Bsrc2pYtCg5FLalwtQbPg7aHS55OY6Q8nl8t+lh1Ryjlm2dvM5BOhnfiCT67I6XhY8IM7FxJHRV0yR4AVZeOUwmbk8+/snZHraTLgWRpvRlVh1lHRxLBe+LkI1q44VPycutzrMzcBmkk3kYsJnNngfXJBT3MfPFyDIIRh+7gZPS+BCUd5PSMKrNSGIRYas48QfqZy8bru39c4t2sNKFrmZrrwNR04Hsu13I2bpdIGWlWPI98ufoWhVsZIvhLauuM/JeNs6wGESqrRB8JBzCZmgVcSHpSj1g7u3JxgvD2RwqtbFahdu5HYVA3TEQnK1hm7YAMsDNsFY+nLF2LTR8f+AFUK3Df7z+XMoV07+QdTEFiog2h0NRvUusapaZ6skb9ouO2V21tZb7qtGXzY5FnTw3pE5X9VjVkgWlfXdultwZXo+ROcjFqxIIRCmaH1/DlKJhmtd0tW4gsaXjlFfJqYwzD6WrA0w4PBS2v5VoG4UadLMtsnZHYxcWjxqXmPbRHi2l50Bqnom1qpefUdhzx/WrsnPZvGj7g3/AryDsAUtDoJn7EhH5Y2ptdGcM8foqaa+YYjolzEuxmZ9rRQ6PZDMg/7qLeOI5S1YrEDQqpNB7W3ZXePXJn3S2QDyornRksNLjy1iBDdgXnnOIYZtSI9hOD0HBtaieekHVEkjBdSy3WZ0uKdlqae98ge2f4uOJhpuZZ8ikRSFmTBvjkBDZZicUWJQgB73468GxGT4BcUbPJQfbQXxJFXKRCMqa+l7fAcrXfNNdcDHRnYdoXxwAU7VY5WiELO9fcNNlV6mZBy/awBHL94uc7oLXV2S3r5HYg6lhhN85kEK1uMicIW6IvxuIEm//87BhQqM/hgsmLHhcJmFbVGTe8FjaIpjxrBG/7mf67W6nlYuuEqnGQMfgZp1hCbsXWRfk6v7630aWPcP2F3pLlao/dN8utkwBFqiTP1s19ux7QqpvU5xGy7m+e8yP8+nrSyZwbMEUUaJX62lonzYO5wtzD0+jESBa60m0i4tNXRVHpmf4Qx+Pc2krGoWlm5AWEjd/1lQEYxtLjRN75M5kF+Do8852yEAAUndFf5Q6QeJhZKzmsYe83Rsdm/h8g422/pnOWk/CGWdEX/aRAfmUToe3stKHE7Cc35vgWQYKqORBL0glhqcg5CLKCLZm+pkLSUjomu1t7VN8y4RgEoa/PpDMi+2HUJoqZwvE8sPWamVYyUA==","attributes":"\u003c?xml version=\"1.0\" encoding=\"utf-8\"?\u003e\n\u003c!DOCTYPE meta SYSTEM \"/SysConfig/Classify/FTImages/classify.dtd\"\u003e\n\u003cmeta\u003e\n    \u003cpicture\u003e\n        \u003ccaption/\u003e\n        \u003ccategory/\u003e\n        \u003ckeywords/\u003e\n        \u003cadinfo/\u003e\n        \u003cphotographer/\u003e\n        \u003ccopyright_info\u003e\n        \u003ccopyright_statement/\u003e\n        \u003ccopyright_group/\u003e\n        \u003cdistribution_rights/\u003e\n        \u003clegal_status/\u003e\n        \u003c/copyright_info\u003e\n        \u003cweb_information\u003e\n            \u003ccaption/\u003e\n            \u003calt_tag/\u003e\n            \u003conline-source/\u003e\n            \u003cmanual-source/\u003e\n            \u003cDIFTcomWebType\u003egraphic\u003c/DIFTcomWebType\u003e\n        \u003c/web_information\u003e\n        \u003cprovider/\u003e\n        \u003cfilm_type/\u003e\n        \u003cdate_taken/\u003e\n        \u003cdate_received/\u003e\n        \u003creference_num/\u003e\n        \u003croll_num/\u003e\n        \u003cframe_num/\u003e\n        \u003ccheckboxes/\u003e\n        \u003cwarnings/\u003e\n        \u003csecurity/\u003e\n        \u003cembargo_date/\u003e\n        \u003cjob_description\u003e\n            \u003ccity/\u003e\n            \u003cprovince/\u003e\n            \u003ccountry/\u003e\n            \u003cinstructions/\u003e\n        \u003c/job_description\u003e\n        \u003cprice/\u003e\n        \u003cfilename/\u003e\n        \u003cbasket/\u003e\n        \u003csource/\u003e\n        \u003cbyline/\u003e\n        \u003cheadline/\u003e\n        \u003cuuid\u003ee7a3b814-59ee-459e-8f60-517f3e80ed99\u003c/uuid\u003e\n        \u003cratio\u003e1.0\u003c/ratio\u003e\n        \u003cimageType\u003eCharts\u003c/imageType\u003e\n    \u003c/picture\u003e\n    \u003cmarkDeleted\u003eFalse\u003c/markDeleted\u003e\n\u003c/meta\u003e\n","workflowStatus":"","systemAttributes":"\u003cprops\u003e\n    \u003cproductInfo\u003e\n        \u003cname\u003eFinancial Times\u003c/name\u003e\n        \u003cissueDate\u003e20151019\u003c/issueDate\u003e\n    \u003c/productInfo\u003e\n    \u003cworkFolder\u003e/FT\u003c/workFolder\u003e\n    \u003csummary/\u003e\n    \u003cimageInfo\u003e\n        \u003cwidth\u003e302\u003c/width\u003e\n        \u003cheight\u003e282\u003c/height\u003e\n        \u003cptWidth\u003e302.0\u003c/ptWidth\u003e\n        \u003cptHeight\u003e282.0\u003c/ptHeight\u003e\n        \u003cxDim\u003e106.54\u003c/xDim\u003e\n        \u003cyDim\u003e99.48\u003c/yDim\u003e\n        \u003cdim\u003e10.654cm x 9.948cm\u003c/dim\u003e\n        \u003cxres\u003e72.0\u003c/xres\u003e\n        \u003cyres\u003e72.0\u003c/yres\u003e\n        \u003ccolorType\u003eRGB\u003c/colorType\u003e\n        \u003cfileType\u003ePNG\u003c/fileType\u003e\n        \u003calphaChannels\u003e1\u003c/alphaChannels\u003e\n    \u003c/imageInfo\u003e\n\u003c/props\u003e\n","usageTickets":"\u003c?xml version='1.0' encoding='UTF-8'?\u003e\n\u003ctl\u003e\n    \u003ct\u003e\n        \u003cid\u003e1\u003c/id\u003e\n        \u003ctp\u003ePublisher\u003c/tp\u003e\n        \u003cc\u003etassellt\u003c/c\u003e\n        \u003ccd\u003e20151019093025\u003c/cd\u003e\n        \u003cdt\u003e\n            \u003cpublishedDate\u003eMon Oct 19 09:30:25 UTC 2015\u003c/publishedDate\u003e\n        \u003c/dt\u003e\n    \u003c/t\u003e\n    \u003ct\u003e\n        \u003cid\u003e2\u003c/id\u003e\n        \u003ctp\u003eweb_publication\u003c/tp\u003e\n        \u003cc\u003etassellt\u003c/c\u003e\n        \u003ccd\u003e20151019093025\u003c/cd\u003e\n        \u003cdt\u003e\n            \u003cwebpublish\u003e\n                \u003csite_url\u003ehttp://www.ft.com/cms/s/e7a3b814-59ee-459e-8f60-517f3e80ed99.html\u003c/site_url\u003e\n                \u003csynd_url\u003ehttp://www.ft.com/cms/s/e7a3b814-59ee-459e-8f60-517f3e80ed99,s01=1.html\u003c/synd_url\u003e\n            \u003c/webpublish\u003e\n        \u003c/dt\u003e\n    \u003c/t\u003e\n    \u003ct\u003e\n        \u003cid\u003e3\u003c/id\u003e\n        \u003ctp\u003eWebCopy\u003c/tp\u003e\n        \u003cc\u003etassellt\u003c/c\u003e\n        \u003ccd\u003e20151019093025\u003c/cd\u003e\n        \u003cdt\u003e\n            \u003crep\u003ecms@ftcmr01-uvpr-uk-p\u003c/rep\u003e\n            \u003cfirst\u003e20151019093025\u003c/first\u003e\n            \u003clast\u003e20151019093025\u003c/last\u003e\n            \u003ccount\u003e1\u003c/count\u003e\n            \u003cchannel\u003eFTcom\u003c/channel\u003e\n        \u003c/dt\u003e\n    \u003c/t\u003e\n    \u003ct\u003e\n        \u003cid\u003e4\u003c/id\u003e\n        \u003ctp\u003emms\u003c/tp\u003e\n        \u003cc\u003eservlet-mms\u003c/c\u003e\n        \u003ccd\u003e20150616131500\u003c/cd\u003e\n        \u003cdt\u003e\n            \u003csrcUUID\u003ee7a3b814-59ee-459e-8f60-517f3e80ed99\u003c/srcUUID\u003e\n            \u003ctrgUUID\u003ee7a3b814-59ee-459e-8f60-517f3e80ed99\u003c/trgUUID\u003e\n            \u003csrcRepo\u003ePROD-cms-read\u003c/srcRepo\u003e\n            \u003ctrgRepo\u003ePROD-cma-write\u003c/trgRepo\u003e\n            \u003cts\u003e1234567890\u003c/ts\u003e\n            \u003ccls\u003ecom.eidosmedia.mms.task.extendedobjectmigrationtask.object.ExtendedObjectImpl\u003c/cls\u003e\n            \u003cseq\u003earchive_on_publish_optimised\u003c/seq\u003e\n            \u003cjob\u003earchive_on_publish_optimised_job\u003c/job\u003e\n        \u003c/dt\u003e\n    \u003c/t\u003e\n\u003c/tl\u003e\n","linkedObjects":[]}`

func TestConstructBodyMessageSuccess(t *testing.T) {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	assert := assert.New(t)
	incomingMsg := consumer.Message{
		map[string]string{
			"Message-Id":        "c4b96810-03e8-4057-84c5-dcc3a8c61a26",
			"Message-Timestamp": "2015-10-19T09:30:29.110Z",
			"Message-Type":      "cms-content-published",
			"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
			"Content-Type":      "application/json",
			"X-Request-Id":      "SYNTHETIC-REQ-MON_Unv1K838lY",
		},
		testBodyRawMsgValue,
	}

	actual, _ := constructMessageBody(incomingMsg, "nativeIngesterTest")

	assert.Equal("2015-10-19T09:30:29.110Z", actual["lastModified"], "Didn't get the expected lastModifiedValue")
}

func TestConstructBodyMessageError(t *testing.T) {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	assert := assert.New(t)
	incomingMsg := consumer.Message{
		map[string]string{
			"Message-Id":        "c4b96810-03e8-4057-84c5-dcc3a8c61a26",
			"Message-Timestamp": "2015-10-19T09:30:29.110Z",
			"Message-Type":      "cms-content-published",
			"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
			"Content-Type":      "application/json",
			"X-Request-Id":      "SYNTHETIC-REQ-MON_Unv1K838lY",
		},
		"foobar",
	}

	_, err := constructMessageBody(incomingMsg, "nativeIngesterTest")

	assert.NotNil(t, err)
}

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
		actualUuid := extractUuid(contents, test.paths)
		if actualUuid != test.uuid {
			t.Errorf("Error extracting uuid.\nHeader: %s\nExpected: %s\nActual: %s\n", test.msgBody, test.uuid, actualUuid)
		}
	}
}
