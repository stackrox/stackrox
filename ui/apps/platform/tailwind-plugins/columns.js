const map = require('lodash/map');

module.exports = function getColumnClasses({ index = {}, variants = ['responsive'] }) {
    return function f({ e, addUtilities }) {
        addUtilities(
            [
                ...map(index, (value, name) => ({
                    [`.${e(`columns-${name}`)}`]: { columnCount: value },
                    [`.${e(`columns-gap-${name}`)}`]: { columnGap: value },
                })),
            ],
            variants
        );
    };
};
