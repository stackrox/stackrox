const map = require('lodash/map');

module.exports = function getOrderClasses({
    positive = {},
    negative = {},
    variants = ['responsive'],
}) {
    return function f({ e, addUtilities }) {
        addUtilities(
            [
                ...map(positive, (value, name) => ({
                    [`.${e(`order-${name}`)}`]: { order: value },
                })),

                ...map(negative, (value, name) => ({
                    [`.${e(`-order${name}`)}`]: { order: value },
                })),
            ],
            variants
        );
    };
};
