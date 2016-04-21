package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/theplant/assettube"
)

func init() {
	assettube.SetConfig(assettube.Config{
		Fingerprint:          true,
		URLPrefix:            "assets",
		SubresourceIntegrity: true,
	})
	assettube.Add("assets")
}

func main() {
	var tmpl = template.New("")
	tmpl.Funcs(template.FuncMap{
		"asset_path": assettube.AssetPath,
		"integrity":  assettube.Integrity,
	})
	tmpl.Parse(`<!DOCTYPE html>
<html>
<head>
	<title>Assetstube</title>
	<link rel="stylesheet" type="text/css" href="{{asset_path "css/app.css"}}">
	<script type="text/javascript" src="{{asset_path "js/app.js"}}" integrity="{{integrity "js/app.js"}}"></script>
</head>
<body>

</body>
</html>
`)

	tmpl.Execute(os.Stdout, nil)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})
	http.HandleFunc("/assets/", assettube.ServeHTTP)
	http.ListenAndServe(":8080", nil)
}
