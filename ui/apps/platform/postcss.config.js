const tailwindcss = require('tailwindcss');

module.exports = {
    plugins: [
        {
            // If Tailwind is entirely removed, this plugin should be removed as well.
            postcssPlugin: 'conditional-tailwindcss',
            async Once(root, { result }) {
                const filePath = result?.opts?.from ?? '';
                const isTailwind = filePath.includes('app.tw.css');

                // Only apply tailwind postcss plugin to our single tailwind entry file.
                // Running tailwind plugin (particularly purge) on every css file
                // causes large memory spikes and slow build times.
                if (!isTailwind) {
                    return;
                }

                const tailwindPlugin = tailwindcss('./tailwind.config.js');

                // Run all tailwind internal plugins sequentially
                await tailwindPlugin.plugins.reduce(
                    (chain, plugin) => chain.then(() => plugin(root, result)),
                    Promise.resolve()
                );
            },
        },
    ],
};
