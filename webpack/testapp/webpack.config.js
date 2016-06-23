var path = require('path');
var webpack = require('webpack');
var ManifestPlugin = require('../index');

var config = {
    entry: {
        app: './index.js',
        app2: './index2.js',
        app3: './src/app3.js'
    },
    output: {
        path: path.join(__dirname, './output'),
        filename: '[name].[chunkhash].js',
        publicPath: './public'
    },
    cache: false,
    // devtool: 'sourcemap',
    plugins: [
        new webpack.optimize.DedupePlugin(),
        new webpack.DefinePlugin({
            'process.env.NODE_ENV': '"production"'
        }),
        new webpack.optimize.UglifyJsPlugin(),
        new webpack.optimize.OccurenceOrderPlugin(),
        new webpack.optimize.AggressiveMergingPlugin(),
        new webpack.NoErrorsPlugin(),
        new webpack.ProvidePlugin({
            'fetch': 'imports?this=>global!exports?global.fetch!whatwg-fetch'
        }),
        new ManifestPlugin({
        	// basePath: 'output/'
        })
    ]
};

module.exports = config;
