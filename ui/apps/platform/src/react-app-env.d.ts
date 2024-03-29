/// <reference types="react-scripts" />

// this overrides the analytics definition thats found in the @types/segment-analytics package to include undefined
// var appears to be required to augment the type because of how its defined in the package
declare var analytics: SegmentAnalytics.AnalyticsJS | undefined; // eslint-disable-line no-var

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
