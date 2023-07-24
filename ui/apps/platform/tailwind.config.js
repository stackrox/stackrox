const config = require('@stackrox/tailwind-config');

const { theme } = config;

module.exports = {
    ...config,
    theme: {
        ...theme,
        fontSize: {
            '2xs': '0.625rem', // 10px
            xs: '0.6875rem', // 11px
            sm: '0.75rem', // 12px
            base: '0.875rem', // 14px
            lg: '1rem', // 16px
            xl: '1.125rem', // 18px
            '2xl': '1.25rem', // 20px
            '3xl': '1.5rem', // 24px
            '4xl': '1.75rem', // 28px
            '5xl': '1.875rem', // 30px
            '6xl': '2.5rem', // 40px
        },
    },
    purge: ['./src/**/*.{js,ts,tsx}'],
};
