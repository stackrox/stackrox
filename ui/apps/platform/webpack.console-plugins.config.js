const path = require('path');
const { ConsoleRemotePlugin } = require('@openshift-console/dynamic-plugin-sdk-webpack');

const config = {
    mode: 'development',
    // No regular entry points. The remote container entry is handled by ConsoleRemotePlugin.
    entry: {},
    context: path.resolve(__dirname, 'src'),
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: '[name]-bundle.js',
        chunkFilename: '[name]-chunk.js',
    },
    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.jsx'],
        modules: ['node_modules', 'src'],
    },
    module: {
        rules: [
            {
                test: /\.(jsx?|tsx?)$/,
                exclude: {
                    and: [/node_modules/],
                    not: [/node_modules\/@lifeomic/],
                },
                use: [
                    {
                        loader: 'ts-loader',
                        options: {
                            configFile: path.resolve(__dirname, 'tsconfig.json'),
                            allowTsInNodeModules: true,
                        },
                    },
                ],
            },
            {
                test: /\.css$/,
                use: ['style-loader', 'css-loader'],
            },
            {
                test: /\.(png|jpg|jpeg|gif|svg|woff2?|ttf|eot|otf)(\?.*$|$)/,
                type: 'asset/resource',
                generator: {
                    filename: 'assets/[name].[ext]',
                },
            },
            {
                test: /\.m?js/,
                resolve: {
                    fullySpecified: false,
                },
            },
        ],
    },
    devServer: {
        static: './dist',
        port: 9001,
        // Allow bridge running in a container to connect to the plugin dev server.
        allowedHosts: 'all',
        headers: {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
            'Access-Control-Allow-Headers': 'X-Requested-With, Content-Type, Authorization',
        },
        devMiddleware: {
            writeToDisk: true,
        },
    },
    plugins: [new ConsoleRemotePlugin()],
    devtool: 'source-map',
    optimization: {
        chunkIds: 'named',
        minimize: false,
    },
};

if (process.env.NODE_ENV === 'production') {
    config.mode = 'production';
    config.output.filename = '[name]-bundle-[hash].min.js';
    config.output.chunkFilename = '[name]-chunk-[chunkhash].min.js';
    config.optimization.chunkIds = 'deterministic';
    config.optimization.minimize = true;
}

module.exports = config;
