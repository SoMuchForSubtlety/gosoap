package gosoap

import (
	"context"
	"crypto/tls"
	"net/http"
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
		_, err := NewClient(sct.URL, nil)
		if err != nil && sct.Err {
			t.Errorf("URL: %s - error: %s", sct.URL, err)
		}
	}
}

func TestSoapClientWithClient(t *testing.T) {
	t.Parallel()
	client, err := NewClient(scts[3].URL, &Config{
		Client: scts[3].Client,
	})

	if client.httpClient != scts[3].Client {
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
	loc, err := time.LoadLocation("CET")
	require.NoError(t, err)
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
				RequestDate: time.Now().In(loc).Format("2006-01-02-07:00"),
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
			client, err := NewClient(tc.wsdl, &Config{LogRequests: true})
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
	assert.ErrorContains(t, err, "wsdl definitions not found")
}

func TestClient_Call_NonUtf8(t *testing.T) {
	t.Skip("server is down")
	t.Parallel()
	soap, err := NewClient("https://demo.ilias.de/webservice/soap/server.php?wsdl", nil)
	assert.NoError(t, err)

	_, err = soap.Call(context.Background(), "login", Params{"client": "demo", "username": "robert", "password": "iliasdemo"})
	assert.NoError(t, err)
}
