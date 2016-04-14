# AssetTube

AssetTube is a tool fingerprints and serves asset files automatically, in runtime.

## How it works

AssetTube will copy your asset files into a subdirectory named `assetstube` and fingerprint it, in runtime. Every time the server is restarted, it will remove previously generated files.

You could check out the [example](https://github.com/theplant/assetstube/tree/master/example) to have better idea of how it works.

## Usage example

```go
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
	<title>AssetTube</title>
	<link rel="stylesheet" type="text/css" href="{{assets "css/app.css"}}">
	<script type="text/javascript" src="{{assets "js/app.js"}}"></script>
</head>
<body>

</body>
</html>`)

	tmpl.Execute(os.Stdout, nil)
}
```
