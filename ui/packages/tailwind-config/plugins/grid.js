const map = require('lodash/map');
const range = require('lodash/range');
const max = require('lodash/max');

module.exports = function getGridClasses({
    grids = range(1, 12),
    gaps = {},
    variants = ['responsive'],
}) {
    return function f({ e, addUtilities }) {
        addUtilities(
            [
                { '.grid': { display: 'grid' } },
                { '.grid-dense': { gridAutoFlow: 'dense' } },
                { '.s-full': { gridColumn: '1 / -1' } },
                ...map(gaps, (size, name) => ({
                    [`.${e(`grid-gap-${name}`)}`]: { gridGap: size },
                })),
                {
                    [`.grid-auto-fit`]: {
                        gridTemplateColumns: `repeat(auto-fit, minmax(var(--min-tile-width, 183px), 1fr))`,
                        gridAutoRows: `minmax(var(--min-tile-height, 180px), auto)`,
                    },
                },
                {
                    [`.grid-auto-fit-wide`]: {
                        gridTemplateColumns: `repeat(auto-fit, minmax(var(--min-tile-width, 230px), 1fr))`,
                        gridAutoRows: `minmax(var(--min-tile-height, 180px), auto)`,
                        maxWidth: `2060px`,
                        marginLeft: `auto`,
                        marginRight: `auto`,
                    },
                },
                ...grids.map((columns) => ({
                    [`.grid-columns-${columns}`]: {
                        gridTemplateColumns: `repeat(${columns}, minmax(var(--min-tile-width, 180px), 1fr))`,
                        gridAutoRows: `minmax(var(--min-tile-height, 180px), auto)`,
                    },
                })),
                ...range(1, max(grids) + 1).map((span) => ({
                    [`.s-${span}`]: {
                        gridColumnStart: `span ${span}`,
                        gridRowEnd: `span ${span}`,
                    },
                    [`.sx-${span}`]: {
                        gridColumnStart: `span ${span}`,
                    },
                    [`.sy-${span}`]: {
                        gridRowEnd: `span ${span}`,
                    },
                })),
            ],
            variants
        );
    };
};
