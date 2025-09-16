import type { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { isSearchCategoryWithFilter } from 'hooks/useAnalytics';
import type {
    AnalyticsEvent,
    NODE_CVE_FILTER_APPLIED,
    PLATFORM_CVE_FILTER_APPLIED,
    WORKLOAD_CVE_FILTER_APPLIED,
    POLICY_VIOLATIONS_FILTER_APPLIED,
} from 'hooks/useAnalytics';

type FilterAppliedEvent =
    | typeof WORKLOAD_CVE_FILTER_APPLIED
    | typeof NODE_CVE_FILTER_APPLIED
    | typeof PLATFORM_CVE_FILTER_APPLIED
    | typeof POLICY_VIOLATIONS_FILTER_APPLIED;

export function createFilterTracker(analyticsTrack: (analyticsEvent: AnalyticsEvent) => void) {
    return function trackAppliedFilter(event: FilterAppliedEvent, payload?: OnSearchPayload) {
        if (!payload || payload.action !== 'ADD') {
            // Only track when a filter is applied, not removed
            return;
        }
        const { category, value: filter } = payload;

        const telemetryEvent = isSearchCategoryWithFilter(category)
            ? { event, properties: { category, filter } }
            : { event, properties: { category } };

        analyticsTrack(telemetryEvent);
    };
}
