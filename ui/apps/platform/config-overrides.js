const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = {
    webpack: function override(config) {
        config.plugins.push(
            new MonacoWebpackPlugin({
                languages: ['json', 'yaml', 'shell'],
            })
        );
        return config;
    },
};
