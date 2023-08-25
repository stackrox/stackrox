/// <reference types="react-scripts" />

// this overrides the analytics definition thats found in the @types/segment-analytics package to include undefined
// var appears to be required to augment the type because of how its defined in the package
declare var analytics: SegmentAnalytics.AnalyticsJS | undefined; // eslint-disable-line no-var

declare module '*.ico' {
    const src: string;
    export default src;
}
