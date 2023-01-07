package receive

const (
	receivePageBaseTemplate = `{{ define "oneshot" }}<!DOCTYPE html>
<html>
<head>
<link rel="apple-touch-icon" href="/assets/icon.png">
<link rel="icon" type="image/png" href="/assets/icon.png">
</head>
<body>
{{ if .FileSection }}{{ template "file-section" . }}{{ end }}
{{ if .InputSection }}{{ if .FileSection }}
<br/>OR<br/>{{ end }}
{{ template "input-section" . }}{{ end }}
</body>
</html>{{ end }}`

	receivePageFileSectionTemplate = `{{ define "file-section" }}<form id="file-form" action="/" method="post" enctype="multipart/form-data">
  {{ if ne .CSRFToken "" }}<input type="hidden" name="csrf-token" value="{{ .CSRFToken }}">{{ end }}
  <h5>Select a file to upload:</h5>
  <input type="file" name="oneshot">
  <br><br>
  <input type="submit" value="Upload">
</form>
{{ if .WithJS }}<script>
  const formElement = document.getElementById("file-form");
  formElement.addEventListener("submit", function(event) {
      event.preventDefault();
      event.stopPropagation();

      const formData = new FormData(formElement);
      const request = new XMLHttpRequest();
      var lengths = [];

      for (const pair of formData.entries()) {
        const name = pair[1].name;
        const size = pair[1].size;
        lengths.push(name + "=" + size.toString()); 
      }

      request.open("POST", "/");
      request.setRequestHeader("X-Oneshot-Multipart-Content-Lengths", lengths.join(";"));
      request.send(formData);
  });
</script>{{ end }}
{{ end }}`

	receivePageInputSectionTemplate = `{{ define "input-section" }}<form action="/" method="post">
  {{ if ne .CSRFToken "" }}<input type="hidden" name="csrf-token" value="{{ .CSRFToken }}">{{ end }}
  <h5>Enter text to send: </h5>
  <textarea name="text"></textarea>
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`
)
