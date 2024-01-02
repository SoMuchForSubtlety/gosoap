package gosoap

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	vatSource = SourceFromURI("https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl")
)

func TestInvalidRequests(t *testing.T) {
	t.Parallel()
	soap, err := NewClient(vatSource, &Config{
		LogRequests: true,
	})
	assert.NoError(t, err)

	cases := []struct {
		name   string
		params any
		err    string
	}{
		{
			name:   "map",
			params: Params{"": ""},
			err:    "xml: start tag with no name",
		},
		{
			name:   "array",
			params: ArrayParams{{"", ""}},
			err:    "xml: start tag with no name",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var err error
			_, err = soap.Call(context.Background(), "checkVat", tc.params)
			assert.EqualError(t, err, tc.err)
		})
	}
}

type TestHeader struct {
	XMLName xml.Name `xml:"TestHeader"`
	Value1  string   `xml:"Value1"`
	Value2  int      `xml:"Value2"`
}

type TestHeader2 struct {
	XMLName xml.Name `xml:"TestHeader2"`
	Value3  string   `xml:"Value3"`
}

func TestEncoding(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		body           any
		headers        []any
		expectedAction string
		expectedBody   string
		config         Config
	}{
		{
			name:           "simple",
			body:           Params{"sIp": "127.0.0.1"},
			expectedAction: "http://lavasoft.com/GetIpLocation",
			expectedBody: `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <soap:Body>
        <GetIpLocation xmlns="http://lavasoft.com/">
            <sIp>127.0.0.1</sIp>
        </GetIpLocation>
    </soap:Body>
</soap:Envelope>`,
		},
		{
			name: "header",
			body: Params{"sIp": "127.0.0.1"},
			headers: []any{
				TestHeader{
					Value1: "testing",
					Value2: 123,
				},
				TestHeader2{
					Value3: "aaa",
				},
			},
			expectedAction: "http://lavasoft.com/GetIpLocation",
			expectedBody: `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <soap:Header xmlns="http://lavasoft.com/">
        <TestHeader>
            <Value1>testing</Value1>
            <Value2>123</Value2>
        </TestHeader>
        <TestHeader2>
            <Value3>aaa</Value3>
        </TestHeader2>
    </soap:Header>
    <soap:Body>
        <GetIpLocation xmlns="http://lavasoft.com/">
            <sIp>127.0.0.1</sIp>
        </GetIpLocation>
    </soap:Body>
</soap:Envelope>`,
		},
		{
			name:    "customized envelope",
			body:    Params{"sIp": "127.0.0.1"},
			headers: []any{Params{"h": "value"}},
			config: Config{
				EnvelopePrefix: "custom",
				EnvelopeAttrs:  map[string]string{"test": "param"},
			},
			expectedAction: "http://lavasoft.com/GetIpLocation",
			expectedBody: `<custom:Envelope test="param">
    <custom:Header xmlns="http://lavasoft.com/">
        <h>value</h>
    </custom:Header>
    <custom:Body>
        <GetIpLocation xmlns="http://lavasoft.com/">
            <sIp>127.0.0.1</sIp>
        </GetIpLocation>
    </custom:Body>
</custom:Envelope>`,
		},

		{
			name:           "auto action",
			body:           Params{"sIp": "127.0.0.1"},
			config:         Config{AutoAction: true},
			expectedAction: "http://lavasoft.com/GeoIPService/GetIpLocation",
			expectedBody: `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <soap:Body>
        <GetIpLocation xmlns="http://lavasoft.com/">
            <sIp>127.0.0.1</sIp>
        </GetIpLocation>
    </soap:Body>
</soap:Envelope>`,
		},
	}

	var reqBody []byte
	var header http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		reqBody = body
		header = r.Header
		resp := `<?xml version="1.0" encoding="utf-8"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
				<soap:Body>
					<m:Response xmlns:m="http://www.test.com/soap/">
						<m:status>OK</m:status>
					</m:Response>
				</soap:Body>
			</soap:Envelope>`
		_, err = w.Write([]byte(resp))
		require.NoError(t, err)
	}))
	defer server.Close()

	spec, err := os.ReadFile("./testdata/ipservice.wsdl")
	require.NoError(t, err)
	spec = bytes.ReplaceAll(spec, []byte("http://wsgeoip.lavasoft.com/ipservice.asmx"), []byte(server.URL))

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			config.Client = server.Client()
			config.Service = "GeoIPService"
			config.Port = "GeoIPServiceSoap"
			client, err := NewClient(SourceFromBytes(spec), &config)
			require.NoError(t, err)
			_, err = client.Call(context.Background(), "GetIpLocation", tc.body, tc.headers...)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(reqBody))
			assert.Equal(t, tc.expectedAction, header["Soapaction"][0])
		})
	}
}
