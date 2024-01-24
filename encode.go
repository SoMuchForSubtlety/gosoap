package gosoap

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"sort"
)

// MarshalXML envelope the body and encode to xml
func (p *process) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	segments := &tokenData{}

	segments.startEnvelope(p.config)

	if p.request.HeaderEntries != nil {
		segments.startHeader(p.namespace, p.config)
		segments.recursiveEncode(p.request.HeaderEntries)
		segments.endHeader(p.config)
	}

	err := segments.startBody(p.request.WSDLOperation, p.namespace, p.config)
	if err != nil {
		return err
	}

	segments.recursiveEncode(p.request.Body)

	// end envelope
	segments.endBody(p.request.WSDLOperation, p.config)
	segments.endEnvelope(p.config)

	for _, t := range segments.data {
		var err error
		if t.token != nil {
			err = e.EncodeToken(t.token)
		} else if t.value != nil {
			err = e.Encode(t.value)
		} else {
			err = e.EncodeElement(t.namedValue.Value(), t.namedValue.Name())
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

type NamedElement interface {
	Name() xml.StartElement
	Value() any
}

func Named(element any, name string) named {
	return named{
		content: element,
		start:   xml.StartElement{Name: xml.Name{Local: name}},
	}
}

type named struct {
	content any
	start   xml.StartElement
}

func (n named) Name() xml.StartElement {
	return n.start
}
func (n named) Value() any {
	return n.content
}

type segment struct {
	token      xml.Token
	value      any
	namedValue NamedElement
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
	case reflect.Struct, reflect.Pointer:
		if named, ok := hm.(NamedElement); ok {
			tokens.data = append(tokens.data, segment{namedValue: named})
		} else {
			tokens.data = append(tokens.data, segment{value: hm})
		}
	}
}

func (tokens *tokenData) startEnvelope(c *Config) {
	e := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", c.EnvelopePrefix),
		},
	}

	e.Attr = make([]xml.Attr, len(c.EnvelopeAttrs))
	for local, value := range c.EnvelopeAttrs {
		e.Attr = append(e.Attr, xml.Attr{
			Name:  xml.Name{Space: "", Local: local},
			Value: value,
		})
	}

	// sort so get a deterministic order (for testing)
	sort.Slice(e.Attr, func(i, j int) bool {
		return e.Attr[i].Name.Local < e.Attr[j].Name.Local
	})

	tokens.data = append(tokens.data, segment{token: e})
}

func (tokens *tokenData) endEnvelope(c *Config) {
	e := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", c.EnvelopePrefix),
		},
	}

	tokens.data = append(tokens.data, segment{token: e})
}

func (tokens *tokenData) startHeader(namespace string, c *Config) {
	h := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", c.EnvelopePrefix),
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: namespace},
		},
	}

	tokens.data = append(tokens.data, segment{token: h})
}

func (tokens *tokenData) endHeader(c *Config) {
	h := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", c.EnvelopePrefix),
		},
	}

	tokens.data = append(tokens.data, segment{token: h})
}

func (tokens *tokenData) startBody(wsdlOperation, namespace string, c *Config) error {
	b := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", c.EnvelopePrefix),
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
func (tokens *tokenData) endBody(wsdlOperation string, c *Config) {
	b := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", c.EnvelopePrefix),
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
