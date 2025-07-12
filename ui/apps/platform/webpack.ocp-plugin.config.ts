/* eslint-env node */

import * as path from 'path';
import { Configuration as WebpackConfiguration } from 'webpack';
import { Configuration as WebpackDevServerConfiguration } from 'webpack-dev-server';
import { ConsoleRemotePlugin } from '@openshift-console/dynamic-plugin-sdk-webpack';

const CopyWebpackPlugin = require('copy-webpack-plugin');

const isProd = process.env.NODE_ENV === 'production';

interface Configuration extends WebpackConfiguration {
    devServer?: WebpackDevServerConfiguration;
}

const config: Configuration = {
    mode: isProd ? 'production' : 'development',
    // No regular entry points needed. All plugin related scripts are generated via ConsoleRemotePlugin.
    entry: {},
    context: path.resolve(__dirname, 'src'),
    output: {
        path: path.resolve(__dirname, 'build', 'ocp-plugin'),
        filename: isProd ? '[name]-bundle-[hash].min.js' : '[name]-bundle.js',
        chunkFilename: isProd ? '[name]-chunk-[chunkhash].min.js' : '[name]-chunk.js',
    },
    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.jsx'],
    },
    module: {
        rules: [
            {
                test: /\.(jsx?|tsx?)$/,
                exclude: /\/node_modules\//,
                use: [
                    {
                        loader: 'ts-loader',
                        options: {
                            configFile: path.resolve(__dirname, 'tsconfig.ocp-plugin.json'),
                        },
                    },
                ],
            },
            {
                test: /\.(css)$/,
                use: ['style-loader', 'css-loader'],
            },
            {
                test: /\.(png|jpg|jpeg|gif|svg|woff2?|ttf|eot|otf)(\?.*$|$)/,
                type: 'asset/resource',
                generator: {
                    filename: isProd ? 'assets/[contenthash][ext]' : 'assets/[name][ext]',
                },
            },
            {
                test: /\.(m?js)$/,
                resolve: {
                    fullySpecified: false,
                },
            },
        ],
    },
    devServer: {
        static: './dist',
        port: 9001,
        // Allow Bridge running in a container to connect to the plugin dev server.
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
    plugins: [
        new ConsoleRemotePlugin(),
        new CopyWebpackPlugin({
            patterns: [
                {
                    from: path.resolve(__dirname, 'locales'),
                    to: 'locales',
                    noErrorOnMissing: true,
                },
            ],
        }),
    ],
    devtool: isProd ? false : 'source-map',
    optimization: {
        chunkIds: isProd ? 'deterministic' : 'named',
        minimize: isProd,
    },
};

export default config;
