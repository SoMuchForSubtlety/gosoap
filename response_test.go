package gosoap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description   string
		response      *Response
		decodeStruct  any
		expectedFault Fault
		expectedError string
	}{
		{
			description: "case: fault error",
			response: &Response{
				Body: []byte(`
				<soap:Fault>
					<faultcode>soap:Server</faultcode>
					<faultstring>Qube.Mama.SoapException: The remote server returned an error: (550) File unavailable (e.g., file not found, no access). The remote server returned an error: (550) File unavailable (e.g., file not found, no access).</faultstring>
					<detail>
					</detail>
			</soap:Fault>	
				`),
			},
			decodeStruct: &struct{}{},
			expectedFault: Fault{
				Code:        "soap:Server",
				Description: "Qube.Mama.SoapException: The remote server returned an error: (550) File unavailable (e.g., file not found, no access). The remote server returned an error: (550) File unavailable (e.g., file not found, no access).",
			},
		},
		{
			description: "case: unmarshal error",
			response: &Response{
				Body: []byte(`
					<GetJobsByIdsResponse
						xmlns="http://webservices.qubecinema.com/XP/Usher/2009-09-29/">
						<GetJobsByIdsResult>
							<JobInfo>
								<ID>9e7d58d9-6f62-43e3-b189-5b1b58eea629</ID>
								<Status>Completed</Status>
								<Progress>0</Progress>
								<VerificationProgress>0</VerificationProgress>
								<EstimatedCompletionTime>0</EstimatedCompletionTime>
							</JobInfo>
						</GetJobsByIdsResult>
					</GetJobsByIdsResponse>
			
				`),
			},
			decodeStruct: &struct {
				XMLName            xml.Name `xml:"GetJobsByIdsResponse"`
				GetJobsByIDsResult string
			}{},
		},
		{
			description: "case: nil error",
			response: &Response{
				Body: []byte(`
					<GetJobsByIdsResponse
						xmlns="http://webservices.qubecinema.com/XP/Usher/2009-09-29/">
						<GetJobsByIdsResult>
							<JobInfo>
								<ID>9e7d58d9-6f62-43e3-b189-5b1b58eea629</ID>
								<Status>Completed</Status>
								<Progress>0</Progress>
								<VerificationProgress>0</VerificationProgress>
								<EstimatedCompletionTime>0</EstimatedCompletionTime>
							</JobInfo>
						</GetJobsByIdsResult>
					</GetJobsByIdsResponse>
			
				`),
			},
			decodeStruct: &struct {
				XMLName            xml.Name `xml:"GetJobsByIdsResponse"`
				GetJobsByIDsResult string
			}{},
		},
		{
			description: "case: body is empty",
			response: &Response{
				Body: []byte(``),
			},
			decodeStruct: &struct {
				XMLName            xml.Name `xml:"GetJobsByIdsResponse"`
				GetJobsByIDsResult string
			}{},
			expectedError: "body is empty",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.description, func(t *testing.T) {
			t.Parallel()
			err := testCase.response.Unmarshal(testCase.decodeStruct)
			if testCase.expectedFault.Code != "" {
				assert.True(t, errors.As(err, &FaultError{}), "should be a fault error")
				assert.ErrorIs(t, err, FaultError{Fault: testCase.expectedFault})
			} else if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsFault(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description          string
		err                  error
		expectedIsFaultError bool
	}{
		{
			description: "case: fault error",
			err: FaultError{
				Fault: Fault{
					Code: "SOAP-ENV:Client",
				},
			},
			expectedIsFaultError: true,
		},
		{
			description:          "case: unmarshal error",
			err:                  fmt.Errorf("unmarshall err: .."),
			expectedIsFaultError: false,
		},
		{
			description:          "case: nil error",
			err:                  nil,
			expectedIsFaultError: false,
		},
	}

	for _, testCase := range testCases {
		t.Logf("running %v test case", testCase.description)

		assert.Equal(t, testCase.expectedIsFaultError, errors.As(testCase.err, &FaultError{}))
	}
}
