const _ = require('lodash');

module.exports = function({ grids = _.range(1, 12), gaps = {}, variants = ['responsive'] }) {
    return function({ e, addUtilities }) {
        addUtilities(
            [
                { '.grid': { display: 'grid' } },
                { '.grid-dense': { gridAutoFlow: 'dense' } },
                { '.s-full': { gridColumn: '1 / -1' } },
                ..._.map(gaps, (size, name) => ({
                    [`.${e(`grid-gap-${name}`)}`]: { gridGap: size }
                })),
                {
                    [`.grid-auto-fit`]: {
                        gridTemplateColumns: `repeat(auto-fit, minmax(var(--min-tile-width, 192px), 1fr))`,
                        gridAutoRows: `minmax(var(--min-tile-height, 180px), auto)`
                    }
                },
                {
                    [`.grid-auto-fit-wide`]: {
                        gridTemplateColumns: `repeat(auto-fit, minmax(var(--min-tile-width, 230px), 1fr))`,
                        gridAutoRows: `minmax(var(--min-tile-height, 180px), auto)`,
                        maxWidth: `2570px`,
                        marginLeft: `auto`,
                        marginRight: `auto`
                    }
                },
                ...grids.map(columns => ({
                    [`.grid-columns-${columns}`]: {
                        gridTemplateColumns: `repeat(${columns}, minmax(var(--min-tile-width, 180px), 1fr))`,
                        gridAutoRows: `minmax(var(--min-tile-height, 180px), auto)`
                    }
                })),
                ..._.range(1, _.max(grids) + 1).map(span => ({
                    [`.s-${span}`]: {
                        gridColumnStart: `span ${span}`,
                        gridRowEnd: `span ${span}`
                    },
                    [`.sx-${span}`]: {
                        gridColumnStart: `span ${span}`
                    },
                    [`.sy-${span}`]: {
                        gridRowEnd: `span ${span}`
                    }
                }))
            ],
            variants
        );
    };
};
