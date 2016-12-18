package browse

import (
	"html/template"
)

var browseTemplate *template.Template

const browseTemplateStr string = `
{{$path := .Path}}
<html>
	<head>
		<script src="/static/js/jquery.min.js"></script>
		<script src="/static/js/bootstrap.min.js"></script>

		<link rel="stylesheet" href="/static/css/bootstrap-custom.css">
	</head>
	<body>
		<ul>
		{{range .DirEnts}}
		<li>
		{{if .IsDir}}
			<a href="/browse/{{$path}}{{.Name}}/">{{.Name}}/</a> <a href="/zip/{{$path}}{{.Name}}/">(zip)</a>
		{{else}}
			<a href="/raw/{{$path}}{{.Name}}">{{.Name}}</a>
		{{end}}
		</li>
		{{end}}
		</ul>
	</body>
</html>
`

func init() {
	browseTemplate = template.Must(template.New("browseTemplate").Parse(browseTemplateStr))
}
