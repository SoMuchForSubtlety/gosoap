package gosoap

import (
	"encoding/xml"
	"fmt"
	"reflect"
)

var (
	soapPrefix                            = "soap"
	customEnvelopeAttrs map[string]string = nil
)

// SetCustomEnvelope define customized envelope
func SetCustomEnvelope(prefix string, attrs map[string]string) {
	soapPrefix = prefix
	if attrs != nil {
		customEnvelopeAttrs = attrs
	}
}

// MarshalXML envelope the body and encode to xml
func (p *process) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	segments := &tokenData{}

	// start envelope
	if p.Client.Definitions == nil {
		return fmt.Errorf("definitions is nil")
	}

	namespace := ""
	if p.Client.Definitions.Types != nil {
		schema := p.Client.Definitions.Types[0].XsdSchema[0]
		namespace = schema.TargetNamespace
		if namespace == "" && len(schema.Imports) > 0 {
			namespace = schema.Imports[0].Namespace
		}
	}
	segments.startEnvelope()

	if p.Request.HeaderEntries != nil {
		segments.startHeader(namespace)
		segments.recursiveEncode(p.Request.HeaderEntries)
		segments.endHeader()
	}

	err := segments.startBody(p.Request.WSDLOperation, namespace)
	if err != nil {
		return err
	}

	segments.recursiveEncode(p.Request.Body)

	// end envelope
	segments.endBody(p.Request.WSDLOperation)
	segments.endEnvelope()

	for _, t := range segments.data {
		var err error
		if t.token != nil {
			err = e.EncodeToken(t.token)
		} else {
			err = e.Encode(t.value)
		}
		if err != nil {
			return err
		}
	}

	return e.Flush()
}

type tokenData struct {
	data []segment
}

type segment struct {
	token xml.Token
	value any
}

func (tokens *tokenData) recursiveEncode(hm any) {
	v := reflect.ValueOf(hm)

	switch v.Kind() {
	case reflect.Map:
		for _, key := range v.MapKeys() {
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: key.String(),
				},
			}

			tokens.data = append(tokens.data, segment{token: t})
			tokens.recursiveEncode(v.MapIndex(key).Interface())
			tokens.data = append(tokens.data, segment{token: xml.EndElement{Name: t.Name}})
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			tokens.recursiveEncode(v.Index(i).Interface())
		}
	case reflect.Array:
		if v.Len() == 2 {
			label := v.Index(0).Interface()
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: label.(string),
				},
			}

			tokens.data = append(tokens.data, segment{token: t})
			tokens.recursiveEncode(v.Index(1).Interface())
			tokens.data = append(tokens.data, segment{token: xml.EndElement{Name: t.Name}})
		}
	case reflect.String:
		content := xml.CharData(v.String())
		tokens.data = append(tokens.data, segment{token: content})
	case reflect.Struct:
		tokens.data = append(tokens.data, segment{value: hm})
	}
}

func (tokens *tokenData) startEnvelope() {
	e := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", soapPrefix),
		},
	}

	if customEnvelopeAttrs == nil {
		e.Attr = []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns:xsi"}, Value: "http://www.w3.org/2001/XMLSchema-instance"},
			{Name: xml.Name{Space: "", Local: "xmlns:xsd"}, Value: "http://www.w3.org/2001/XMLSchema"},
			{Name: xml.Name{Space: "", Local: "xmlns:soap"}, Value: "http://schemas.xmlsoap.org/soap/envelope/"},
		}
	} else {
		e.Attr = make([]xml.Attr, 0)
		for local, value := range customEnvelopeAttrs {
			e.Attr = append(e.Attr, xml.Attr{
				Name:  xml.Name{Space: "", Local: local},
				Value: value,
			})
		}
	}

	tokens.data = append(tokens.data, segment{token: e})
}

func (tokens *tokenData) endEnvelope() {
	e := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", soapPrefix),
		},
	}

	tokens.data = append(tokens.data, segment{token: e})
}

func (tokens *tokenData) startHeader(namespace string) {
	h := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", soapPrefix),
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: namespace},
		},
	}

	tokens.data = append(tokens.data, segment{token: h})
}

func (tokens *tokenData) endHeader() {
	h := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", soapPrefix),
		},
	}

	tokens.data = append(tokens.data, segment{token: h})
}

func (tokens *tokenData) startBody(wsdlOperation, namespace string) error {
	b := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", soapPrefix),
		},
	}

	if wsdlOperation == "" {
		return fmt.Errorf("operation is empty")
	} else if namespace == "" {
		return fmt.Errorf("namespace is empty")
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: wsdlOperation,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: namespace},
		},
	}

	tokens.data = append(tokens.data, segment{token: b}, segment{token: r})

	return nil
}

// endToken close body of the envelope
func (tokens *tokenData) endBody(wsdlOperation string) {
	b := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", soapPrefix),
		},
	}

	r := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: wsdlOperation,
		},
	}

	tokens.data = append(tokens.data, segment{token: r}, segment{token: b})
}
