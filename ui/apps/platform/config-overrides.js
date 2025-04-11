module.exports = {
    jest: function override(config) {
        /* eslint-disable no-param-reassign */
        config.transform['^.+\\.css$'] = '<rootDir>/react-app-rewired/jest/cssTransform.js';
        config.transform['^(?!.*\\.(js|jsx|mjs|cjs|ts|tsx|css|json)$)'] =
            '<rootDir>/react-app-rewired/jest/fileTransform.js';
        /* eslint-enable no-param-reassign */
        return config;
    },
    webpack: function override(config) {
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
