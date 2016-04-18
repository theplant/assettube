package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/theplant/assettube"
)

func init() {
	assettube.Add("assets")
}

func main() {
	var tmpl = template.New("")
	tmpl.Funcs(template.FuncMap{
		"assets": assettube.AssetsPath,
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

	http.HandleFunc("/assets", assettube.ServeHTTP)
}
