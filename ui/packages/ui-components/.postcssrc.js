/* eslint @typescript-eslint/no-var-requires: 0 */

const tailwindcss = require('tailwindcss');

module.exports = {
    plugins: [tailwindcss('./tailwind.config.js')],
};
