package template

import (
	"embed"
	"html/template"
	"io"
	"strings"
)

//go:generate make webrtc-client
var (
	//go:embed webrtc-client.js
	ClientJS template.JS

	//go:embed sd-streams-polyfill.min.js
	PolyfillJS template.JS

	indentFunc = func(spaces int, v template.HTML) template.HTML {
		if v == "" {
			return v
		}
		vs := string(v)
		pad := strings.Repeat(" ", spaces)
		return template.HTML(pad + strings.Replace(vs, "\n", "\n"+pad, -1) + "\n")
	}

	//go:embed templates/*.html
	tmpltFS embed.FS
	tmplt   = template.Must(
		template.New("root").
			Funcs(template.FuncMap{
				"indent": indentFunc,
			}).
			ParseFS(tmpltFS, "templates/*.html"),
	)
)

func init() {
	if len(ClientJS) == 0 {
		panic("browserClientJS is empty")
	}

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

	Head       template.HTML
	ClientJS   template.JS
	PolyfillJS template.JS
}

func WriteTo(w io.Writer, ctx Context) error {
	return tmplt.ExecuteTemplate(w, "index", ctx)
}
