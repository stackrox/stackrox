const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = {
    jest: function override(config) {
        config.transform['^.+\\.css$'] = '<rootDir>/react-app-rewired/jest/cssTransform.js';
        config.transform['^(?!.*\\.(js|jsx|mjs|cjs|ts|tsx|css|json)$)'] =
            '<rootDir>/react-app-rewired/jest/fileTransform.js';
        return config;
    },
    webpack: function override(config) {
        config.plugins.push(
            new MonacoWebpackPlugin({
                languages: ['json', 'yaml', 'shell'],
            })
        );
        return config;
    },
};
