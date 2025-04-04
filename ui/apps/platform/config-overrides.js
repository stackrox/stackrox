const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = {
    webpack: function override(config) {
        config.plugins.push(
            new MonacoWebpackPlugin({
                languages: ['json', 'yaml', 'shell'],
            })
        );

        const sourceMapRule = config.module.rules.find((rule) =>
            rule?.loader?.includes('source-map-loader')
        );

        if (sourceMapRule) {
            // Override CRA source map exclusions to exclude @redocly/config and keep
            // the default babel exclusions defined in node_modules/react-scripts/config/webpack.config.js.
            // Redoc >= 2.3.0 throws dev server warnings due to missing source map files.
            sourceMapRule.exclude = /(@babel(?:\/|\\{1,2})runtime)|(@redocly\/config)/;
        }

        return config;
    },
};
