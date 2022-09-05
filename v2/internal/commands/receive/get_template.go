package receive

const (
	receivePageBaseTemplate = `{{ define "base" }}<!DOCTYPE html>
<html>
<head>
<link rel="apple-touch-icon" href="/assets/icon.png">
<link rel="icon" type="image/png" href="/assets/icon.png">
</head>
<body>
{{ if .FileSection }}
  {{ template "file-section" .CSRFToken }}
{{ end }}
{{ if .InputSection }}
  {{ if .FileSection }}
    <br/>OR<br/>
  {{ end }}
  {{ template "input-section" .CSRFToken }}
{{ end }}
</body>
</html>
{{ end }}`

	receivePageFileSectionTemplate = `{{ define "file-section" }}<form action="/" method="post" enctype="multipart/form-data">
  {{ if ne . "" }}<input type="hidden" name="csrf-token" value="{{ . }}">{{ end }}
  <h5>Select a file to upload:</h5>
  <input type="file" name="oneshot">
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`

	receivePageInputSectionTemplate = `{{ define "input-section" }}<form action="/" method="post">
  {{ if ne . "" }}<input type="hidden" name="csrf-token" value="{{ . }}">{{ end }}
  <h5>Enter text to send: </h5>
  <textarea name="text"></textarea>
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`
)
