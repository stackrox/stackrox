/// <reference types="vite/client" />
/// <reference types="vite-plugin-svgr/client" />

declare global {
    // Allows importing of .ico files as a string representing the URL path to the file
    module '*.ico' {
        const src: string;
        export default src;
    }

    namespace React {
        // Extend CSSProperties to allow custom CSS properties
        interface CSSProperties {
            // Adds PatternFly CSS properties
            [key: `--pf-v5-${string}`]: string | number | undefined;
        }
    }
}

export {};
