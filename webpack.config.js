var webpack = require('webpack');

// webpack.config.js
module.exports = {
    entry: "./ui-thing/local.js",
    output: {
        path: __dirname + "/ui-thing/built/",
        publicPath: "/repos/built/",
        filename: "bundle.js"
    },
    module: {
        loaders: [
            { test: /\.css$/, loaders: ['style-loader', 'css-loader'] },
            { test: /\.(woff2?|ttf|eot|svg)$/, loader: 'url-loader?limit=10000' },
        ],
    },
    plugins: [
        new webpack.ProvidePlugin({
            $: "jquery",
            jQuery: "jquery",
            Cookies: "js-cookie"
       })
    ]
};
