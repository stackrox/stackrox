import { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import {
    AnalyticsEvent,
    NODE_CVE_FILTER_APPLIED,
    PLATFORM_CVE_FILTER_APPLIED,
    WORKLOAD_CVE_FILTER_APPLIED,
    isSearchCategoryWithFilter,
} from 'hooks/useAnalytics';

type FilterAppliedEvent =
    | typeof WORKLOAD_CVE_FILTER_APPLIED
    | typeof NODE_CVE_FILTER_APPLIED
    | typeof PLATFORM_CVE_FILTER_APPLIED;

export function createFilterTracker(analyticsTrack: (analyticsEvent: AnalyticsEvent) => void) {
    return function trackAppliedFilter(event: FilterAppliedEvent, payload: OnSearchPayload) {
        const { action, category, value: filter } = payload;

        if (action !== 'ADD') {
            // Only track when a filter is applied, not removed
            return;
        }

        const telemetryEvent = isSearchCategoryWithFilter(category)
            ? { event, properties: { category, filter } }
            : { event, properties: { category } };

        analyticsTrack(telemetryEvent);
    };
}
