package template

import (
	"embed"
	"html/template"
	"io"
)

var (
	//go:generate make webrtc-client
	//go:embed webrtc-client.js
	ClientJS template.HTML

	//go:embed templates/*.html
	tmpltFS embed.FS
	tmplt   = template.Must(
		template.New("root").ParseFS(tmpltFS, "templates/*.html"),
	)
)

func init() {
	if len(ClientJS) == 0 {
		panic("browserClientJS is empty")
	}
	ClientJS = "<script>\n" + ClientJS + "\n</script>"

	if tmplt == nil {
		panic("tmplt is nil")
	}
}

type Context struct {
	AutoConnect bool

	RTCConfigJSON  string
	OfferJSON      string
	Endpoint       string
	BasicAuthToken string
	SessionToken   string

	Head     template.HTML
	ClientJS template.HTML
}

func WriteTo(w io.Writer, ctx Context) error {
	return tmplt.ExecuteTemplate(w, "index", ctx)
}
