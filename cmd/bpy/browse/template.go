package browse

import (
	"html/template"
)

var browseTemplate *template.Template

const browseTemplateStr string = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">

    <title>buppy - {{$.Path}}</title>

    <link rel="stylesheet" href="/static/css/bootstrap-custom.css">
    <link rel="stylesheet" href="/static/css/common.css">
    <link rel="stylesheet" href="/static/css/browse.css">
  </head>

  <body>
    <nav class="navbar navbar-default navbar-fixed-top">
      <div class="container">

        <div class="col-md-8 col-md-offset-2">
          <ol class="breadcrumb">
            <li><a href="/browse/"><b>buppy</b></a></li>
            {{range .PathParts}}
            <li><a href="../">{{.}}</a></li>
            {{end}}
          </ol>
        </div>
        
      </div>
    </nav>

    <div class="container">

      <div class="row features">
        <div class="col-md-8 col-md-offset-2">
          <ul class="nav nav-pills folder-actions">

            <li><a href="../" class="btn btn-borderless">
              <span class="glyphicon glyphicon-arrow-up"></span>
              <span class="hidden-xs">Parent</span>
            </a></li>

            <li><a href="/zip/{{$.Path}}" class="btn btn-borderless">
              <span class="glyphicon glyphicon-save"></span>
              <span class="hidden-xs">Download</span>
            </a></li>

          </ul>

          <h1>{{$.Name}}</h1>
        </div>

        <div class="col-md-8 col-md-offset-2">
          <table class="table table-hover">
            <colgroup>
              <col><col><col><col><col>
            </colgroup>
            <tbody>
              {{range .DirEnts}}
              <tr>
                <td>
                  {{if .IsDir}}
                  <a href="/browse/{{$.Path}}{{.Name}}/" title="{{.Name}}"><span class="glyphicon glyphicon-folder-open"></span>{{.Name}}</a>
                  {{else}}
                  <a href="/raw/{{$.Path}}{{.Name}}" title="{{.Name}}"><span class="glyphicon glyphicon-file"></span>{{.Name}}</a>
                  {{end}}
                </td>
                <td class="text-muted text-monospace" title="{{.Mode}}"><small>{{.Mode}}</small></td>
                <td class="text-muted" title="{{.ModTime}}"><small>{{.ModTime}}</small></td>
                <td class="text-muted text-monospace" title="{{.Size}} bytes">
                  {{if .IsDir}}{{else}}
                  <small>{{.Size}} B</small>
                  {{end}}
                </td>
                <td>
                  {{if .IsDir}}
                  <a href="/zip/{{$.Path}}{{.Name}}/" title="Download zipped folder" download><span class="glyphicon glyphicon-save"></span></a>
                  {{else}}
                  <a href="/raw/{{$.Path}}{{.Name}}" title="Download file" download><span class="glyphicon glyphicon-save"></span></a>
                  {{end}}
                </td>
              </tr>
		          {{end}}
            </tbody>
          </table>
        </div>
      </div>
      
    </div>

  </body>
</html>`

func init() {
	browseTemplate = template.Must(template.New("browseTemplate").Parse(browseTemplateStr))
}
