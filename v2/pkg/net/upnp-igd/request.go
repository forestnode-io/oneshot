package upnpigd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type request struct {
	url    string
	body   string
	header http.Header

	service  string
	function string
}

func (r *request) do(ctx context.Context, client *http.Client) ([]byte, error) {
	payload := strings.NewReader(envelope(r.body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	addSOAPRequestHeaders(r.header, r.service, r.function)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http call to service: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	_ = resp.Body.Close()

	if 400 <= resp.StatusCode {
		return nil, fmt.Errorf("got error status code from service: %d %s", resp.StatusCode, resp.Status)
	}

	return body, nil
}

func addSOAPRequestHeaders(h http.Header, service, function string) {
	h.Set("Content-Type", `text/xml; charset="utf-8"`)
	h["SOAPAction"] = []string{fmt.Sprintf(`"%s#%s"`, service, function)}
	h.Set("Connection", "close")
	h.Set("Cache-Control", "no-cache")
	h.Set("Pragma", "no-cache")
}

func envelope(payload string) string {
	tmplt := `<?xml version="1.0"?>
	<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"
	s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
	<s:Body>%s</s:Body>
	</s:Envelope>
`

	return fmt.Sprintf(tmplt, payload)
}
