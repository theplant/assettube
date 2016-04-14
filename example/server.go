package main

import (
	"html/template"
	"os"

	"github.com/theplant/assetstube"
)

func init() {
	assetstube.Add("assets")
}

func main() {
	var tmpl = template.New("")
	tmpl.Funcs(template.FuncMap{
		"assets": assetstube.AssetsPath,
	})
	tmpl.Parse(`<!DOCTYPE html>
<html>
<head>
	<title>Assetstube</title>
	<link rel="stylesheet" type="text/css" href="{{assets "css/app.css"}}">
	<script type="text/javascript" src="{{assets "js/app.js"}}"></script>
</head>
<body>

</body>
</html>`)

	tmpl.Execute(os.Stdout, nil)
}
