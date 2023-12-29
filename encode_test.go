package gosoap

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidRequests(t *testing.T) {
	t.Parallel()
	soap, err := SoapClientWithConfig("https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl", nil, &Config{
		Dump: true,
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

func TestSetCustomEnvelope(t *testing.T) {
	t.Parallel()
	SetCustomEnvelope("soapenv", map[string]string{
		"xmlns:soapenv": "http://schemas.xmlsoap.org/soap/envelope/",
		"xmlns:tem":     "http://tempuri.org/",
	})

	// TODO: actual test
	_, err := SoapClient("https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl", nil)
	assert.NoError(t, err)
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

func TestClient_Headers(t *testing.T) {
	t.Parallel()
	soap, err := SoapClient("https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl", nil)
	assert.NoError(t, err)

	p := process{
		Client: soap,
		Request: &Request{
			WSDLOperation: "aaaaa",
			HeaderEntries: []any{
				TestHeader{
					Value1: "test",
					Value2: 123,
				},
				TestHeader2{
					Value3: ":)",
				},
			},
		},
	}

	var resultBuf bytes.Buffer
	err = p.MarshalXML(xml.NewEncoder(bufio.NewWriter(&resultBuf)), xml.StartElement{})
	assert.NoError(t, err)
	fmt.Println(resultBuf.String())
}

func TestClient_HeaderArray(t *testing.T) {
	t.Parallel()
	soap, err := SoapClient("https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl", nil)
	assert.NoError(t, err)

	p := process{
		Client: soap,
		Request: &Request{
			WSDLOperation: "wsdlOp",
			HeaderEntries: []any{
				TestHeader{
					Value1: "test",
					Value2: 123,
				},
				TestHeader2{
					Value3: ":)",
				},
			},
		},
	}

	var resultBuf bytes.Buffer
	err = p.MarshalXML(xml.NewEncoder(bufio.NewWriter(&resultBuf)), xml.StartElement{})
	assert.NoError(t, err)
	fmt.Println(resultBuf.String())
}
