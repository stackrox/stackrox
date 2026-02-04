import type { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { payloadItemFiltererForTracking } from 'Components/CompoundSearchFilter/utils/utils';
import { isSearchCategoryWithFilter } from 'hooks/useAnalytics';
import type {
    AnalyticsEvent,
    NODE_CVE_FILTER_APPLIED,
    PLATFORM_CVE_FILTER_APPLIED,
    POLICY_VIOLATIONS_FILTER_APPLIED,
    VIEW_BASED_REPORT_FILTER_APPLIED,
    WORKLOAD_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';

type FilterAppliedEvent =
    | typeof WORKLOAD_CVE_FILTER_APPLIED
    | typeof NODE_CVE_FILTER_APPLIED
    | typeof PLATFORM_CVE_FILTER_APPLIED
    | typeof POLICY_VIOLATIONS_FILTER_APPLIED
    | typeof VIEW_BASED_REPORT_FILTER_APPLIED;

export function createFilterTracker(analyticsTrack: (analyticsEvent: AnalyticsEvent) => void) {
    return function trackAppliedFilter(event: FilterAppliedEvent, payload?: OnSearchPayload) {
        if (Array.isArray(payload)) {
            payload.filter(payloadItemFiltererForTracking).forEach((payloadItem) => {
                const { category, value: filter } = payloadItem;

                // TODO do 'SELECT_INCLUSIVE' and 'SELECT_EXCLUSIVE' actions require allow list?
                const telemetryEvent = isSearchCategoryWithFilter(category)
                    ? { event, properties: { category, filter } }
                    : { event, properties: { category } };

                analyticsTrack(telemetryEvent);
            });
        }
    };
}
