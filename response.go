package gosoap

import (
	"encoding/xml"
	"fmt"
)

// Response Soap Response
type Response struct {
	// see https://www.w3.org/TR/2000/NOTE-SOAP-20000508/#_Toc478383503
	Body []byte
	// see https://www.w3.org/TR/2000/NOTE-SOAP-20000508/#_Toc478383497
	HeaderEntries []byte
}

// FaultError implements error interface
type FaultError struct {
	Fault Fault
}

func (e FaultError) Error() string {
	return e.Fault.String()
}

func (e FaultError) Is(target error) bool {
	f, ok := target.(FaultError)
	if !ok {
		return false
	}

	return f.Fault.Code == e.Fault.Code && f.Fault.Description == e.Fault.Description
}

func (r *Response) Unmarshal(v any) error {
	if len(r.Body) == 0 {
		return fmt.Errorf("body is empty")
	}

	var fault Fault
	err := xml.Unmarshal(r.Body, &fault)
	if err != nil {
		return fmt.Errorf("error unmarshalling the body to Fault: %v", err.Error())
	}
	if fault.Code != "" {
		return FaultError{Fault: fault}
	}

	return xml.Unmarshal(r.Body, v)
}

func (r *Response) UnmarshalHeaders(v any) error {
	if len(r.HeaderEntries) == 0 {
		return fmt.Errorf("Header is empty")
	}

	return xml.Unmarshal(r.HeaderEntries, v)
}
