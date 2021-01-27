module.exports = function getObjectFitClasses({ variants }) {
    return function f({ addUtilities }) {
        addUtilities(
            {
                '.object-contain': { objectFit: 'contain' },
                '.object-cover': { objectFit: 'cover' },
                '.object-fill': { objectFit: 'fill' },
                '.object-none': { objectFit: 'none' },
                '.object-scale': { objectFit: 'scale-down' },
            },
            variants
        );
    };
};
