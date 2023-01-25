/// <reference types="react-scripts" />

export declare global {
    interface Window {
        analytics?: SegmentAnalytics.AnalyticsJS | undefined;
    }

    declare module '*.ico' {
        const src: string;
        export default src;
    }
}
