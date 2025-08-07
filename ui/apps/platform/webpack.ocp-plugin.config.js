const path = require('path');
const fs = require('fs');
const { DefinePlugin } = require('webpack');
const { ConsoleRemotePlugin } = require('@openshift-console/dynamic-plugin-sdk-webpack');
const CopyWebpackPlugin = require('copy-webpack-plugin');

const acsRootBaseUrl = '/acs';

const isProd = process.env.NODE_ENV === 'production';

/*
 * Alias all top level directories and files under `/src/` so that we can import them in our code
 * via `import { SomeComponent } from 'Components/SomeComponent`;`. This mirrors the Vite configuration approach.
 */
function getSrcAliases() {
    const aliases = {};

    fs.readdirSync(path.resolve(__dirname, 'src'), { withFileTypes: true }).forEach(({ name }) => {
        if (name.startsWith('.')) {
            // avoid hidden directories, like `.DS_Store`
            return;
        }
        const alias = name.includes('.') ? name.split('.').slice(0, -1).join('.') : name;
        aliases[alias] = path.resolve(__dirname, 'src', name);
    });

    return aliases;
}

const config = {
    mode: isProd ? 'production' : 'development',
    // No regular entry points needed. All plugin related scripts are generated via ConsoleRemotePlugin.
    entry: {},
    context: path.resolve(__dirname, 'src'),
    output: {
        path: path.resolve(__dirname, 'build', 'static', 'ocp-plugin'),
        filename: isProd ? '[name]-bundle-[hash].min.js' : '[name]-bundle.js',
        chunkFilename: isProd ? '[name]-chunk-[chunkhash].min.js' : '[name]-chunk.js',
    },
    resolve: {
        extensions: ['.js', '.jsx', '.ts', '.tsx'],
        alias: getSrcAliases(),
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
                            transpileOnly: true,
                            configFile: path.resolve(__dirname, 'tsconfig.json'),
                        },
                    },
                ],
            },
            {
                test: /\.(css)$/,
                use: ['style-loader', 'css-loader'],
            },
            {
                test: /\.(png|jpg|jpeg|gif|svg|woff2?|ttf|eot|otf|ico)(\?.*$|$)/,
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
        port: 9001,
        // Allow Bridge running in a container to connect to the plugin dev server.
        allowedHosts: 'all',
        headers: {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
            'Access-Control-Allow-Headers': 'X-Requested-With, Content-Type, Authorization',
        },
        devMiddleware: {
            // The ConsoleRemotePlugin sets a publicPath of '/api/plugins/<plugin-name>/', however when running the
            // console locally in development mode, the proxy strips off this prefix and only leaves '/', which causes
            // the plugin to not be able to find its assets.
            publicPath: '/',
        },
    },
    plugins: [
        new ConsoleRemotePlugin({
            validateSharedModules: false,
            pluginMetadata: {
                name: 'advanced-cluster-security',
                version: '0.0.1',
                displayName: 'Red Hat Advanced Cluster Security for OpenShift',
                description: 'OCP Console Plugin for Advanced Cluster Security',
                exposedModules: {
                    context: './ConsolePlugin/PluginProvider',
                    AdministrationNamespaceSecurityTab:
                        './ConsolePlugin/AdministrationNamespaceSecurityTab/AdministrationNamespaceSecurityTab',
                    CveDetailPage: './ConsolePlugin/CveDetailPage/CveDetailPage',
                    ImageDetailPage: './ConsolePlugin/ImageDetailPage/ImageDetailPage',
                    ProjectSecurityTab: './ConsolePlugin/ProjectSecurityTab/ProjectSecurityTab',
                    SecurityVulnerabilitiesPage:
                        './ConsolePlugin/SecurityVulnerabilitiesPage/SecurityVulnerabilitiesPage',
                    WorkloadSecurityTab: './ConsolePlugin/WorkloadSecurityTab/WorkloadSecurityTab',
                },
                dependencies: {
                    '@console/pluginAPI': '>=4.19.0',
                },
            },
            extensions: [
                // General Context Provider to be shared across all extensions
                {
                    type: 'console.context-provider',
                    properties: {
                        provider: { $codeRef: 'context.PluginProvider' },
                        useValueHook: { $codeRef: 'context.usePluginContext' },
                    },
                },
                // Security Vulnerabilities Page
                {
                    type: 'console.page/route',
                    properties: {
                        exact: true,
                        path: `${acsRootBaseUrl}/security/vulnerabilities`,
                        component: {
                            $codeRef: 'SecurityVulnerabilitiesPage.SecurityVulnerabilitiesPage',
                        },
                    },
                },
                {
                    type: 'console.navigation/section',
                    properties: {
                        id: 'acs-security',
                        name: 'Security',
                        startsWith: `${acsRootBaseUrl}/security`,
                        insertBefore: ['compute', 'usermanagement', 'administration'],
                    },
                },
                {
                    type: 'console.navigation/href',
                    properties: {
                        id: 'security-vulnerabilities',
                        name: 'Vulnerabilities',
                        section: 'acs-security',
                        href: `${acsRootBaseUrl}/security/vulnerabilities`,
                        perspective: 'admin',
                    },
                },
                // Workload Detail Page Security Tab
                ...['Deployment', 'ReplicaSet', 'StatefulSet', 'DaemonSet', 'Job', 'CronJob'].map(
                    (kind) => ({
                        type: 'console.tab/horizontalNav',
                        properties: {
                            model: {
                                group: 'apps',
                                kind,
                                version: 'v1',
                            },
                            page: {
                                name: 'Security',
                                href: 'security',
                            },
                            component: { $codeRef: 'WorkloadSecurityTab.WorkloadSecurityTab' },
                        },
                    })
                ),
                // Administration Namespace Security Tab
                {
                    type: 'console.tab/horizontalNav',
                    properties: {
                        model: {
                            group: '',
                            kind: 'Namespace',
                            version: 'v1',
                        },
                        page: {
                            name: 'Security',
                            href: 'security',
                        },
                        component: {
                            $codeRef:
                                'AdministrationNamespaceSecurityTab.AdministrationNamespaceSecurityTab',
                        },
                    },
                },
                // Project Security Tab
                {
                    type: 'console.tab/horizontalNav',
                    properties: {
                        model: {
                            group: 'project.openshift.io',
                            kind: 'Project',
                            version: 'v1',
                        },
                        page: {
                            name: 'Security',
                            href: 'security',
                        },
                        component: {
                            $codeRef: 'ProjectSecurityTab.ProjectSecurityTab',
                        },
                    },
                },
                // Image Detail Page
                {
                    type: 'console.page/route',
                    properties: {
                        exact: true,
                        path: `${acsRootBaseUrl}/security/vulnerabilities/images/:imageId`,
                        component: { $codeRef: 'ImageDetailPage.ImageDetailPage' },
                    },
                },
                // Image CVE Detail Page
                {
                    type: 'console.page/route',
                    properties: {
                        exact: true,
                        path: `${acsRootBaseUrl}/security/vulnerabilities/cves/:cveId`,
                        component: { $codeRef: 'CveDetailPage.CveDetailPage' },
                    },
                },
            ],
        }),
        new CopyWebpackPlugin({
            patterns: [
                {
                    from: path.resolve(__dirname, 'locales'),
                    to: 'locales',
                    noErrorOnMissing: true,
                },
            ],
        }),
        new DefinePlugin({
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
            'process.env.ACS_CONSOLE_DEV_TOKEN': JSON.stringify(
                // Do not inject the token when building for production
                process.env.NODE_ENV === 'development'
                    ? process.env.ACS_CONSOLE_DEV_TOKEN
                    : undefined
            ),
        }),
    ],
    devtool: isProd ? false : 'source-map',
    optimization: {
        chunkIds: isProd ? 'deterministic' : 'named',
        minimize: isProd,
    },
};

module.exports = config;
