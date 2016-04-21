package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/theplant/assettube"
)

func init() {
	assettube.SetFingerprint(true)
	assettube.SetURLPrefix("/assets")
	assettube.Add("assets")
}

func main() {
	var tmpl = template.New("")
	tmpl.Funcs(template.FuncMap{
		"asset_path": assettube.AssetPath,
	})
	tmpl.Parse(`<!DOCTYPE html>
<html>
<head>
	<title>Assetstube</title>
	<link rel="stylesheet" type="text/css" href="{{asset_path "css/app.css"}}">
	<script type="text/javascript" src="{{asset_path "js/app.js"}}"></script>
</head>
<body>

</body>
</html>`)

	tmpl.Execute(os.Stdout, nil)

	http.HandleFunc("/assets/", assettube.ServeHTTP)
	http.ListenAndServe(":8080", nil)
}
