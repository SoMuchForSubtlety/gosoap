package gosoap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getWsdlBody(t *testing.T) {
	t.Parallel()
	type args struct {
		u string
	}
	dir, _ := os.Getwd()

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				u: "https://[::1]:namedport",
			},
			wantErr: true,
		},
		{
			args: args{
				u: filepath.Join(dir, "testdata/ipservice.wsdl"),
			},
			wantErr: true,
		},
		{
			args: args{
				u: "file://" + filepath.Join(dir, "testdata/ipservice.wsdl"),
			},
			wantErr: false,
		},
		{
			args: args{
				u: "file:",
			},
			wantErr: true,
		},
		{
			args: args{
				u: "https://www.google.com/",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := getWsdlBody(tt.args.u, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("getwsdlBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestFaultString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		description      string
		fault            *Fault
		expectedFaultStr string
	}{
		{
			description: "success case: fault string",
			fault: &Fault{
				Code:        "soap:SERVER",
				Description: "soap exception",
				Detail:      "soap detail",
			},
			expectedFaultStr: "[soap:SERVER]: soap exception | Detail: soap detail",
		},
	}

	for _, testCase := range testCases {
		t.Logf("running %v testCase", testCase.description)

		faultStr := testCase.fault.String()
		assert.Equal(t, testCase.expectedFaultStr, faultStr)
	}
}
