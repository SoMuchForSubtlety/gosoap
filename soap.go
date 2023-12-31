package gosoap

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/net/html/charset"
)

// Params type is used to set the params in soap request
type Params map[string]any

type (
	ArrayParams [][2]any
	SliceParams []any
)

// Config config the Client
type Config struct {
	Client      *http.Client
	AutoAction  bool
	LogRequests bool
	Logger      CommunicationLogger

	Service string
	Port    string

	EnvelopePrefix string
	EnvelopeAttrs  map[string]string

	Username string
	Password string
}

// NewClient return new *Client to handle the requests with the WSDL
func NewClient(wsdlSource WSDLSource, config *Config) (*Client, error) {
	if config == nil {
		config = &Config{}
	}

	if config.Client == nil {
		config.Client = http.DefaultClient
	}

	if config.Logger == nil {
		// default to info because the default logger has debug logs disabled
		config.Logger = NewSlogAdapter(slog.Default(), slog.LevelInfo)
	}

	if config.EnvelopePrefix == "" {
		config.EnvelopePrefix = "soap"
	}
	if len(config.EnvelopeAttrs) == 0 {
		config.EnvelopeAttrs = map[string]string{
			"xmlns:xsi":  "http://www.w3.org/2001/XMLSchema-instance",
			"xmlns:xsd":  "http://www.w3.org/2001/XMLSchema",
			"xmlns:soap": "http://schemas.xmlsoap.org/soap/envelope/",
		}
	}

	definitions, err := getWSDLDefinitions(wsdlSource, config)
	if err != nil {
		return nil, err
	}

	var namespace string
	if definitions.Types != nil {
		// FIXME: this can be incorrect, we need to add a config option to select a type element (by targetNamespace?)
		schema := definitions.Types[0].XsdSchema[0]
		namespace = schema.TargetNamespace
		if namespace == "" && len(schema.Imports) > 0 {
			namespace = schema.Imports[0].Namespace
		}
	}

	service, port, err := definitions.serviceAndPort(config.Service, config.Port)
	if err != nil {
		return nil, fmt.Errorf("could not determin SOAP address: %w", err)
	}
	config.Service = service.Name
	config.Port = port.Name

	bindingName := port.Binding
	splitBindingName := strings.Split(port.Binding, ":")
	if len(splitBindingName) == 2 {
		bindingName = splitBindingName[1] // strip off the namespace
	}
	var binding *wsdlBinding
	for _, b := range definitions.Bindings {
		if b.Name == bindingName || b.Name == port.Binding {
			binding = b
			break
		}
	}
	if binding == nil {
		return nil, fmt.Errorf("could not find binding matching %q", port.Binding)
	}
	return &Client{
		config:        *config,
		httpClient:    config.Client,
		binding:       binding,
		autoActionURL: strings.TrimSuffix(definitions.TargetNamespace, "/"),
		address:       port.SoapAddresses[0].Location, // TODO: use multiple addresses?
		namespace:     namespace,
	}, nil
}

func (d *wsdlDefinitions) serviceAndPort(serviceName, portName string) (*wsdlService, *wsdlPort, error) {
	var service *wsdlService
	if len(d.Services) == 0 {
		return nil, nil, errors.New("WSDL has no services")
	}
	if serviceName == "" {
		service = d.Services[0]
	} else {
		var possibleServices []string
		for _, svc := range d.Services {
			possibleServices = append(possibleServices, svc.Name)
			if svc.Name == serviceName {
				service = svc
				break
			}
		}
		if service == nil {
			return nil, nil, fmt.Errorf("no service matching %q found, possible values are %v", serviceName, possibleServices)
		}
	}
	if len(service.Ports) == 0 {
		return nil, nil, fmt.Errorf("WSDL service %q has no ports", service.Name)
	}
	var port *wsdlPort
	if portName == "" {
		port = service.Ports[0]
	} else {
		var possiblePorts []string
		for _, p := range service.Ports {
			possiblePorts = append(possiblePorts, p.Name)
			if p.Name == portName {
				port = p
				break
			}
		}
		if port == nil {
			return nil, nil, fmt.Errorf("no port matching %q found, possible values are %v", portName, possiblePorts)
		}
	}
	if len(port.SoapAddresses) == 0 {
		return nil, nil, fmt.Errorf("WSDL port %q has no addresses", port.Name)
	}
	return service, port, nil
}

// Client struct hold all the information about WSDL,
// request and response of the server
type Client struct {
	httpClient *http.Client
	config     Config

	address       string
	namespace     string
	autoActionURL string
	binding       *wsdlBinding
}

func (c *Client) Call(ctx context.Context, wsdlOperation string, body any, headerParams ...any) (res *Response, err error) {
	return c.Do(ctx, NewRequest(wsdlOperation, body, headerParams...))
}

// Do Process Soap Request
func (c *Client) Do(ctx context.Context, req *Request) (res *Response, err error) {
	action, err := c.binding.GetSoapActionFromWsdlOperation(req.WSDLOperation)
	if err != nil {
		return nil, err
	}
	p := &process{
		config:     &c.config,
		namespace:  c.namespace,
		request:    req,
		soapAction: action,
	}

	if p.soapAction == "" && c.config.AutoAction {
		p.soapAction = fmt.Sprintf("%s/%s/%s", c.autoActionURL, c.config.Service, req.WSDLOperation)
	}

	p.payload, err = xml.MarshalIndent(p, "", "    ")
	if err != nil {
		return nil, err
	}

	b, err := c.doRequest(ctx, p)
	if err != nil {
		return nil, ErrorWithPayload{err, p.payload}
	}

	var soap SoapEnvelope
	// err = xml.Unmarshal(b, &soap)
	// error: xml: encoding "ISO-8859-1" declared but Decoder.CharsetReader is nil
	// https://stackoverflow.com/questions/6002619/unmarshal-an-iso-8859-1-xml-input-in-go
	// https://github.com/golang/go/issues/8937

	decoder := xml.NewDecoder(bytes.NewReader(b))
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&soap)

	res = &Response{
		Body:          soap.Body.Contents,
		HeaderEntries: soap.Header.Contents,
	}
	if err != nil {
		return res, ErrorWithPayload{err, p.payload}
	}

	return res, nil
}

type process struct {
	config    *Config
	request   *Request
	namespace string
	// see https://www.w3.org/TR/2000/NOTE-SOAP-20000508/#_Toc478383528
	soapAction string
	payload    []byte
}

// doRequest makes new request to the server using the c.Method, c.URL and the body.
// body is enveloped in Do method
func (c *Client) doRequest(ctx context.Context, p *process) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.address, bytes.NewBuffer(p.payload))
	if err != nil {
		return nil, err
	}

	if c.config.LogRequests {
		var body []byte
		req.Body, body, err = drainBody(req.Body)
		if err != nil {
			return nil, err
		}
		c.config.Logger.LogRequest(p.request.WSDLOperation, req.Header, body)
	}

	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.ContentLength = int64(len(p.payload))

	req.Header.Add("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Add("Accept", "text/xml")
	if p.soapAction != "" {
		req.Header.Add("SOAPAction", p.soapAction)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if c.config.LogRequests {
		var body []byte
		resp.Body, body, err = drainBody(resp.Body)
		if err != nil {
			return nil, err
		}
		c.config.Logger.LogResponse(p.request.WSDLOperation, req.Header, body)
	}

	return io.ReadAll(resp.Body)
}

// from net/http/httputil
func drainBody(b io.ReadCloser) (r1 io.ReadCloser, r2 []byte, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, nil, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, nil, err
	}
	if err = b.Close(); err != nil {
		return nil, nil, err
	}
	return io.NopCloser(&buf), buf.Bytes(), nil
}

// ErrorWithPayload error payload schema
type ErrorWithPayload struct {
	error
	Payload []byte
}

// GetPayloadFromError returns the payload of a ErrorWithPayload
func GetPayloadFromError(err error) []byte {
	if err, ok := err.(ErrorWithPayload); ok {
		return err.Payload
	}
	return nil
}

// SoapEnvelope struct
type SoapEnvelope struct {
	XMLName struct{} `xml:"Envelope"`
	Header  SoapHeader
	Body    SoapBody
}

// SoapHeader struct
type SoapHeader struct {
	XMLName  struct{} `xml:"Header"`
	Contents []byte   `xml:",innerxml"`
}

// SoapBody struct
type SoapBody struct {
	XMLName  struct{} `xml:"Body"`
	Contents []byte   `xml:",innerxml"`
}
