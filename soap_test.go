package gosoap

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var scts = []struct {
	URL    string
	Err    bool
	Client *http.Client
}{
	{
		URL: "://www.server",
		Err: false,
	},
	{
		URL: "",
		Err: false,
	},
	{
		URL: "https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl",
		Err: true,
	},
	{
		URL: "https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl",
		Err: true,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	},
}

func TestSoapClient(t *testing.T) {
	t.Parallel()
	for _, sct := range scts {
		_, err := SoapClient(sct.URL, nil)
		if err != nil && sct.Err {
			t.Errorf("URL: %s - error: %s", sct.URL, err)
		}
	}
}

func TestSoapClientWithClient(t *testing.T) {
	t.Parallel()
	client, err := SoapClient(scts[3].URL, scts[3].Client)

	if client.HTTPClient != scts[3].Client {
		t.Errorf("HTTP client is not the same as in initialization: - error: %s", err)
	}

	if err != nil {
		t.Errorf("URL: %s - error: %s", scts[3].URL, err)
	}
}

type CheckVatRequest struct {
	CountryCode string `xml:"countryCode"`
	VatNumber   string `xml:"vatNumber"`
}

type CheckVatResponse struct {
	CountryCode string `xml:"countryCode"`
	VatNumber   string `xml:"vatNumber"`
	RequestDate string `xml:"requestDate"`
	Valid       string `xml:"valid"`
	Name        string `xml:"name"`
	Address     string `xml:"address"`
}

type CapitalCityResponse struct {
	CapitalCityResult string
}

type NumberToWordsResponse struct {
	NumberToWordsResult string
}

func TestValidRequests(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		wsdl         string
		operation    string
		request      any
		resp         any
		expectedResp any
	}{
		{
			name:      "vat",
			wsdl:      "https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl",
			operation: "checkVat",
			request: Params{
				"vatNumber":   "6388047V",
				"countryCode": "IE",
			},
			resp: &CheckVatResponse{},
			expectedResp: &CheckVatResponse{
				CountryCode: "IE",
				VatNumber:   "6388047V",
				Name:        "GOOGLE IRELAND LIMITED",
				Address:     "3RD FLOOR, GORDON HOUSE, BARROW STREET, DUBLIN 4",
				Valid:       "true",
				RequestDate: time.Now().Format(time.DateOnly) + "+01:00",
			},
		},
		{
			name:      "capital",
			wsdl:      "http://webservices.oorsprong.org/websamples.countryinfo/CountryInfoService.wso?WSDL",
			operation: "CapitalCity",
			request:   Params{"sCountryISOCode": "GB"},
			resp:      &CapitalCityResponse{},
			expectedResp: &CapitalCityResponse{
				CapitalCityResult: "London",
			},
		},
		{
			name:      "numbers",
			wsdl:      "https://www.dataaccess.com/webservicesserver/numberconversion.wso?WSDL",
			operation: "NumberToWords",
			request:   Params{"ubiNum": "23"},
			resp:      &NumberToWordsResponse{},
			expectedResp: &NumberToWordsResponse{
				NumberToWordsResult: "twenty three ",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, err := SoapClientWithConfig(tc.wsdl, nil, &Config{Dump: true})
			require.NoError(t, err)
			res, err := client.Call(context.Background(), tc.operation, tc.request)
			require.NoError(t, err)
			assert.NotNil(t, res)
			err = res.Unmarshal(tc.resp)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResp, tc.resp)
		})
	}
}

func TestInvalidWSDL(t *testing.T) {
	t.Parallel()
	c := &Client{}
	_, err := c.Call(context.Background(), "", Params{})
	assert.ErrorContains(t, err, "unsupported protocol scheme")

	c.SetWSDL("://test.")
	_, err = c.Call(context.Background(), "checkVat", Params{})
	assert.ErrorContains(t, err, "missing protocol scheme")
}

type customLogger struct{}

func (c customLogger) LogRequest(method string, dump []byte) {
	re := regexp.MustCompile(`(<vatNumber>)[\s\S]*?(<\/vatNumber>)`)
	maskedResponse := re.ReplaceAllString(string(dump), `${1}XXX${2}`)

	log.Printf("%s request: %s", method, maskedResponse)
}

func (c customLogger) LogResponse(method string, dump []byte) {
	if method == "checkVat" {
		return
	}

	log.Printf("Response: %s", dump)
}

func TestClient_Call_WithCustomLogger(t *testing.T) {
	t.Parallel()
	soap, err := SoapClientWithConfig("https://ec.europa.eu/taxation_customs/vies/checkVatService.wsdl",
		nil,
		&Config{Dump: true, Logger: &customLogger{}},
	)
	assert.NoError(t, err)

	var res *Response

	res, err = soap.Call(context.Background(), "checkVat", Params{
		"countryCode": "IE",
		"vatNumber":   "6388047V",
	})
	assert.NoError(t, err)

	var rv CheckVatResponse
	err = res.Unmarshal(&rv)
	assert.NoError(t, err)
	if rv.CountryCode != "IE" {
		t.Errorf("error: %+v", rv)
	}
}

func TestClient_Call_NonUtf8(t *testing.T) {
	t.Skip("server is down")
	t.Parallel()
	soap, err := SoapClient("https://demo.ilias.de/webservice/soap/server.php?wsdl", nil)
	assert.NoError(t, err)

	_, err = soap.Call(context.Background(), "login", Params{"client": "demo", "username": "robert", "password": "iliasdemo"})
	assert.NoError(t, err)
}

func TestProcess_doRequest(t *testing.T) {
	t.Parallel()
	c := &process{
		Client: &Client{
			HTTPClient: &http.Client{},
		},
	}

	_, err := c.doRequest(context.Background(), "")
	assert.Error(t, err)

	_, err = c.doRequest(context.Background(), "://teste.")
	assert.Error(t, err)
}
