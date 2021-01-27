module.exports = {
    printWidth: 100,
    singleQuote: true,
    tabWidth: 4,
    overrides: [
        {
            files: '*.css',
            parser: 'css',
        },
        {
            files: '*.json',
            parser: 'json',
        },
        {
            files: '*.md',
            parser: 'markdown',
            options: {
                printWidth: 80,
                proseWrap: 'always',
            },
        },
    ],
};
