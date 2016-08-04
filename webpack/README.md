# AssetTube

Install assettube npm package:

```shell
npm install assettube
```

In your webpack config file:

```js
var path = require('path');
var webpack = require('webpack');
var AssetTube = require('../index');

var config = {
	entry: { app: './index.js'},
	output: {
		path: path.join(__dirname, './output'),
		filename: '[name].[chunkhash].js',
		publicPath: './public'
	},
	plugins: [
		new AssetTube({
			// configurations
			// hostname: '',
			// urlPrefix: '',
			// basePath: '',
			// fileName: 'assettube.json',
			// stripSrc: null,
			// transformExtensions: /^(gz|map)$/i,
			// cache: null
		})
	]
};

module.exports = config;
```

Go server:

```go
m, err := NewManagerManifest("path/to/assettube.json")
if err != nil {
	t.Fatal(err)
}

// the reset same as runtime mode
```
