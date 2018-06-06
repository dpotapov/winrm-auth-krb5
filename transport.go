package winrmkrb5

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dpotapov/go-spnego"
	"github.com/pkg/errors"

	"github.com/masterzen/winrm"
	"github.com/masterzen/winrm/soap"
)

const soapXML = "application/soap+xml"

// Transport implements the winrm.Transporter interface.
type Transport struct {
	HTTPClient *http.Client
	Endpoint   *winrm.Endpoint
}

// Transport applies configuration parameters from the Endpoint to the underlying HTTPClient.
// If the HTTPClient is nil, a new instance of http.Client will be created.
func (t *Transport) Transport(endpoint *winrm.Endpoint) error {
	if t.HTTPClient == nil {
		t.HTTPClient = &http.Client{
			Transport: &spnego.Transport{},
		}
	}
	if httpTr, ok := t.HTTPClient.Transport.(*spnego.Transport); ok {
		if httpTr.TLSClientConfig == nil {
			httpTr.TLSClientConfig = &tls.Config{}
		}
		httpTr.TLSClientConfig.InsecureSkipVerify = endpoint.Insecure
		httpTr.TLSClientConfig.ServerName = endpoint.TLSServerName
		httpTr.ResponseHeaderTimeout = endpoint.Timeout
		if len(endpoint.CACert) > 0 {
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(endpoint.CACert) {
				return errors.New("unable to read certificates")
			}
			httpTr.TLSClientConfig.RootCAs = certPool
		}
	} else {
		return errors.New("unable to apply WinRM endpoint parameters to unknown HTTP Transport")
	}

	t.Endpoint = endpoint
	return nil
}

// Post sends a POST request to WinRM server with the provided SOAP payload.
// If the WinRM web service responds with Unauthorized status, the method performs KRB5 authentication.
func (t *Transport) Post(client *winrm.Client, request *soap.SoapMessage) (string, error) {
	req, err := t.makeRequest(request.String())
	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := t.body(resp)
	if err != nil {
		return "", err
	}

	bodyErrStr := func(body string) string {
		if len(body) > 100 {
			return body[:100] + "..."
		}
		if len(body) == 0 {
			return "<no http content>"
		}
		return body
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http error %d: %s", resp.StatusCode, bodyErrStr(body))
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, soapXML) {
		return body, fmt.Errorf("incorrect Content-Type \"%s\" (expected %s): %s",
			ct, soapXML, bodyErrStr(body))
	}
	return body, nil
}

// EndpointURL returns a WinRM http(s) URL.
// It does the same job as unexported method url() for the winrm.Endpoint type.
func (t *Transport) EndpointURL() string {
	scheme := "http"
	if t.Endpoint.HTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/wsman", scheme, t.Endpoint.Host, t.Endpoint.Port)
}

func (t *Transport) makeRequest(payload string) (*http.Request, error) {
	req, err := http.NewRequest("POST", t.EndpointURL(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", soapXML+";charset=UTF-8")
	return req, nil
}

// body func reads the response body and return it as a string
func (t *Transport) body(response *http.Response) (string, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading http response body")
	}
	return string(body), errors.Wrap(response.Body.Close(), "reading http response body")
}
