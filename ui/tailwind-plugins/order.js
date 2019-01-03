const _ = require("lodash");

module.exports = function({
    positive = {},
    negative = {},
    variants = ["responsive"]
}) {
    return function({ e, addUtilities }) {
        addUtilities(
            [
                ..._.map(positive, (value, name) => ({
                    [`.${e(`order-${name}`)}`]: { order: value }
                })),

                ..._.map(negative, (value, name) => ({
                    [`.${e(`-order${name}`)}`]: { order: value }
                }))
            ],
            variants
        );
    };
};
