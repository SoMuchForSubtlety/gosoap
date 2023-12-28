package gosoap

// A representation of a SOAP Request
type Request struct {
	// wsdl operation name, this will be used to map to a SOAP action
	// https://www.w3.org/TR/2001/NOTE-wsdl-20010315#_soap:operation
	WSDLOperation string
	// see https://www.w3.org/TR/2000/NOTE-SOAP-20000508/#_Toc478383503
	Body any
	// see https://www.w3.org/TR/2000/NOTE-SOAP-20000508/#_Toc478383497
	HeaderEntries []any
}

func NewRequest(wsdlOperation string, body any, headerBlocks ...any) *Request {
	return &Request{
		WSDLOperation: wsdlOperation,
		Body:          body,
		HeaderEntries: headerBlocks,
	}
}
