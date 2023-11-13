/*
----------------------------------------------------------------------------
How to use in markup                               

 <div className={`bg-gradient-horizontal`} style={{'--start': 'var(--primary-300)', '--end': 'var(--primary-600)'}}>

----------------------------------------------------------------------------
*/

module.exports = function getGradientClasses({ variants }) {
    return function f({ addUtilities }) {
        const utilities = {
            [`.bg-gradient-horizontal`]: {
                backgroundImage: `linear-gradient(to right, var(--start, purple), var(--end, blue))`,
            },
            [`.bg-gradient-vertical`]: {
                backgroundImage: `linear-gradient(to bottom, var(--start, purple), var(--end, blue))`,
            },
            [`.bg-gradient-diagonal`]: {
                backgroundImage: `linear-gradient(to bottom right, var(--start, purple), var(--end, blue))`,
            },
            [`.bg-gradient-radial`]: {
                backgroundImage: `radial-gradient(var(--start, purple), var(--end, blue))`,
            },
        };

        addUtilities(utilities, variants);
    };
};
