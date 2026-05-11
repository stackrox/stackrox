const path = require('path');

const webpack = require('webpack');

// Resolve the axe-core path in Node so it can be passed to the browser via Cypress.env('AXE_CORE_PATH')
const axeCorePath = require.resolve('axe-core/axe.min.js');

const outputDir = path.resolve(__dirname, 'cypress/webpack-output');
const compilers = {};

function makeWebpackConfig(filePath) {
    return {
        mode: 'development',
        devtool: 'inline-source-map',
        entry: { [filePath.replace(/[/\\]/g, '_')]: filePath },
        output: { path: outputDir, filename: '[name].js' },
        resolve: { extensions: ['.ts', '.js'] },
        module: {
            rules: [
                {
                    test: /\.[jt]s$/,
                    exclude: /node_modules/,
                    use: {
                        loader: 'ts-loader',
                        options: {
                            transpileOnly: true,
                            configFile: path.resolve(__dirname, 'cypress/tsconfig.json'),
                        },
                    },
                },
            ],
        },
    };
}

function compile(compiler) {
    return new Promise((resolve, reject) => {
        compiler.run((err, stats) => {
            if (err) {
                return reject(err);
            }
            if (stats.hasErrors()) {
                return reject(new Error(stats.toString({ errors: true })));
            }
            return resolve();
        });
    });
}

function webpackPreprocessor(file) {
    const { filePath, shouldWatch } = file;
    const bundlePath = path.join(outputDir, `${filePath.replace(/[/\\]/g, '_')}.js`);

    if (shouldWatch) {
        if (compilers[filePath]) {
            return compilers[filePath].promise;
        }

        const compiler = webpack(makeWebpackConfig(filePath));
        const bundle = { initial: true };
        bundle.promise = new Promise((resolve, reject) => {
            compiler.watch({}, (err, stats) => {
                if (err || stats.hasErrors()) {
                    const error = err || new Error(stats.toString({ errors: true }));
                    reject(error);
                    return;
                }
                resolve(bundlePath);
                bundle.promise = Promise.resolve(bundlePath);
                if (!bundle.initial) {
                    file.emit('rerun');
                }
                bundle.initial = false;
            });
        });

        compilers[filePath] = bundle;

        file.on('close', () => {
            delete compilers[filePath];
            compiler.close(() => {});
        });

        return bundle.promise;
    }

    const compiler = webpack(makeWebpackConfig(filePath));
    return compile(compiler).then(() => {
        compiler.close(() => {});
        return bundlePath;
    });
}

/*
 * The helper function intended to provide automatic code completion for configuration in many popular code editors
 * had subtle side-effect to cause some typescript-eslint/no-unsafe-return errors in unit test files.
 *
 * const { defineConfig } = require('cypress'); // eslint-disable-line import/no-extraneous-dependencies
 * module.exports = defineConfig({ … });
 */

module.exports = {
    chromeWebSecurity: false, // Browser options
    defaultCommandTimeout: 8000, // Timeouts options
    numTestsKeptInMemory: 0, // Global options
    requestTimeout: 20000, // Timeouts options
    video: true, // Videos options
    videoCompression: 32, // Videos options

    retries: {
        // Configure retry attempts for `cypress run`
        // Attempt a single retry for failed tests when run headless
        runMode: 1,
        // Configure retry attempts for `cypress open`
        openMode: 0,
    },

    e2e: {
        baseUrl: 'https://localhost:3000',
        viewportHeight: 850, // Viewport options
        viewportWidth: 1440, // Viewport options
        setupNodeEvents: (on, config) => {
            // eslint-disable-next-line no-param-reassign
            config.env.AXE_CORE_PATH = axeCorePath;
            on('task', {
                beforeSuite(spec) {
                    // eslint-disable-next-line no-console
                    console.log(`${new Date().toISOString()} running test suite: ${spec.name}\n`);
                    return null;
                },
                joinPaths(paths) {
                    return path.join(...paths);
                },
            });
            on('file:preprocessor', webpackPreprocessor);
            return config;
        },
    },

    component: {
        devServer: {
            framework: 'react',
            bundler: 'vite',
        },
        viewportHeight: 600,
        viewportWidth: 800,
    },
};
