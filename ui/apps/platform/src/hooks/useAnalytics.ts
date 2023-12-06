import { useCallback } from 'react';
import { useSelector } from 'react-redux';
import { Telemetry } from 'types/config.proto';

import { selectors } from 'reducers';
import { UnionFrom, tupleTypeGuard } from 'utils/type.utils';

// event name constants
export const CLUSTER_CREATED = 'Cluster Created';
export const INVITE_USERS_MODAL_OPENED = 'Invite Users Modal Opened';
export const INVITE_USERS_SUBMITTED = 'Invite Users Submitted';
export const WATCH_IMAGE_MODAL_OPENED = 'Watch Image Modal Opened';
export const WATCH_IMAGE_SUBMITTED = 'Watch Image Submitted';

export const WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED = 'Workload CVE Entity Context View';
export const WORKLOAD_CVE_FILTER_APPLIED = 'Workload CVE Filter Applied';
export const WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED = 'Workload CVE Default Filters Changed';
export const COLLECTION_CREATED = 'Collection Created';
export const VULNERABILITY_REPORT_CREATED = 'Vulnerability Report Created';
export const VULNERABILITY_REPORT_DOWNLOAD_GENERATED = 'Vulnerability Report Download Generated';
export const VULNERABILITY_REPORT_SENT_MANUALLY = 'Vulnerability Report Sent Manually';
export const GLOBAL_SNOOZE_CVE = 'Global Snooze CVE';

/**
 * Boolean fields should be tracked with 0 or 1 instead of true/false. This
 * allows us to use the boolean fields in numeric aggregations in the
 * analytics dashboard to retrieve an accurate count of the number of times
 * a property was enabled for an event.
 */
type AnalyticsBoolean = 0 | 1;

// search categories and type guards for tracking search filters on the Workload CVE pages
export const searchCategoriesWithFilter = [
    'CVE',
    'IMAGE',
    'COMPONENT',
    'COMPONENT SOURCE',
    'SEVERITY',
    'FIXABLE',
] as const;
export const isSearchCategoryWithFilter = tupleTypeGuard(searchCategoriesWithFilter);
export type SearchCategoryWithFilter = UnionFrom<typeof searchCategoriesWithFilter>;

export const searchCategoriesWithoutFilter = ['DEPLOYMENT', 'NAMESPACE', 'CLUSTER'] as const;
export const isSearchCategoryWithoutFilter = tupleTypeGuard(searchCategoriesWithoutFilter);
export type SearchCategoryWithoutFilter = UnionFrom<typeof searchCategoriesWithoutFilter>;

/**
 * An AnalyticsEvent is either a simple string that represents the event name,
 * or an object with an event name and additional properties.
 */
type AnalyticsEvent =
    | typeof CLUSTER_CREATED
    | typeof INVITE_USERS_MODAL_OPENED
    | typeof INVITE_USERS_SUBMITTED
    /** Tracks each time the user opens the "Watched Images" modal */
    | typeof WATCH_IMAGE_MODAL_OPENED
    /** Tracks each time the user submits a request to watch an image */
    | typeof WATCH_IMAGE_SUBMITTED
    /**
     * Tracks each view of a CVE entity context (CVE, Image, or Deployment). This is
     * controlled by the entity tabs on the Overview page and the CVE Detail page.
     */
    | {
          event: typeof WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED;
          properties: {
              type: 'CVE' | 'Image' | 'Deployment';
              page: 'Overview' | 'CVE Detail';
          };
      }
    /**
     * Tracks each time the user applies a filter on a Workload page.
     * This is controlled by the main search bar on all Workload CVE pages.
     * We only track the value of the applied filter when it does not represent
     * specifics of a customer environment.
     */
    | {
          event: typeof WORKLOAD_CVE_FILTER_APPLIED;
          properties:
              | { category: SearchCategoryWithFilter; filter: string }
              | { category: SearchCategoryWithoutFilter };
      }
    /**
     * Tracks each time the user changes the default filters on the Workload CVE overview page.
     */
    | {
          event: typeof WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED;
          properties: {
              SEVERITY_CRITICAL: AnalyticsBoolean;
              SEVERITY_IMPORTANT: AnalyticsBoolean;
              SEVERITY_MODERATE: AnalyticsBoolean;
              SEVERITY_LOW: AnalyticsBoolean;
              CVE_STATUS_FIXABLE: AnalyticsBoolean;
              CVE_STATUS_NOT_FIXABLE: AnalyticsBoolean;
          };
      }
    /**
     * Tracks each time the user creates a collection.
     */
    | {
          event: typeof COLLECTION_CREATED;
          properties: {
              source: 'Vulnerability Reporting' | 'Collections';
          };
      }
    /**
     *
     */
    | {
          event: typeof VULNERABILITY_REPORT_CREATED;
          properties: {
              SEVERITY_CRITICAL: AnalyticsBoolean;
              SEVERITY_IMPORTANT: AnalyticsBoolean;
              SEVERITY_MODERATE: AnalyticsBoolean;
              SEVERITY_LOW: AnalyticsBoolean;
              CVE_STATUS_FIXABLE: AnalyticsBoolean;
              CVE_STATUS_NOT_FIXABLE: AnalyticsBoolean;
              IMAGE_TYPE_DEPLOYED: AnalyticsBoolean;
              IMAGE_TYPE_WATCHED: AnalyticsBoolean;
              EMAIL_NOTIFIER: AnalyticsBoolean;
              TEMPLATE_MODIFIED: AnalyticsBoolean;
          };
      }
    /**
     * Tracks each time the user generates a vulnerability report download.
     */
    | typeof VULNERABILITY_REPORT_DOWNLOAD_GENERATED
    /**
     * Tracks each time the user sends a vulnerability report manually.
     */
    | typeof VULNERABILITY_REPORT_SENT_MANUALLY
    /**
     * Tracks each time the user snoozes a Node or Platform CVE via
     * Vulnerability Management 1.0
     */
    | {
          event: typeof GLOBAL_SNOOZE_CVE;
          properties: {
              type: 'NODE' | 'PLATFORM';
              cve: string;
              duration: string;
          };
      };

const useAnalytics = () => {
    const telemetry = useSelector(selectors.publicConfigTelemetrySelector);
    const { enabled: isTelemetryEnabled } = telemetry || ({} as Telemetry);

    const analyticsPageVisit = useCallback(
        (type: string, name: string, additionalProperties = {}): void => {
            if (isTelemetryEnabled !== false) {
                window.analytics?.page(type, name, additionalProperties);
            }
        },
        [isTelemetryEnabled]
    );

    const analyticsTrack = useCallback(
        (analyticsEvent: AnalyticsEvent): void => {
            if (isTelemetryEnabled === false) {
                return;
            }

            if (typeof analyticsEvent === 'string') {
                window.analytics?.track(analyticsEvent);
            } else {
                window.analytics?.track(analyticsEvent.event, analyticsEvent.properties);
            }
        },
        [isTelemetryEnabled]
    );

    return { analyticsPageVisit, analyticsTrack };
};

export default useAnalytics;
