import { useCallback } from 'react';
import { useSelector } from 'react-redux';
import Raven from 'raven-js';
import mapValues from 'lodash/mapValues';

import { Telemetry } from 'types/config.proto';
import { selectors } from 'reducers';
import { UnionFrom, ensureExhaustive, tupleTypeGuard } from 'utils/type.utils';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

// Event Name Constants

// clusters
export const CLUSTER_CREATED = 'Cluster Created';

// invite users
export const INVITE_USERS_MODAL_OPENED = 'Invite Users Modal Opened';
export const INVITE_USERS_SUBMITTED = 'Invite Users Submitted';

// network graph
export const CLUSTER_LEVEL_SIMULATOR_OPENED = 'Network Graph: Cluster Level Simulator Opened';
export const GENERATE_NETWORK_POLICIES = 'Network Graph: Generate Network Policies';
export const DOWNLOAD_NETWORK_POLICIES = 'Network Graph: Download Network Policies';
export const CIDR_BLOCK_FORM_OPENED = 'Network Graph: CIDR Block Form Opened';

// watch images
export const WATCH_IMAGE_MODAL_OPENED = 'Watch Image Modal Opened';
export const WATCH_IMAGE_SUBMITTED = 'Watch Image Submitted';

// workload cves
export const WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED = 'Workload CVE Entity Context View';
export const WORKLOAD_CVE_FILTER_APPLIED = 'Workload CVE Filter Applied';
export const WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED = 'Workload CVE Default Filters Changed';
export const WORKLOAD_CVE_DEFERRAL_EXCEPTION_REQUESTED =
    'Workload CVE Deferral Exception Requested';
export const WORKLOAD_CVE_FALSE_POSITIVE_EXCEPTION_REQUESTED =
    'Workload CVE False Positive Exception Requested';
export const COLLECTION_CREATED = 'Collection Created';
export const VULNERABILITY_REPORT_CREATED = 'Vulnerability Report Created';
export const VULNERABILITY_REPORT_DOWNLOAD_GENERATED = 'Vulnerability Report Download Generated';
export const VULNERABILITY_REPORT_SENT_MANUALLY = 'Vulnerability Report Sent Manually';
export const IMAGE_SBOM_GENERATED = 'Image SBOM Generated';

// node and platform CVEs
export const GLOBAL_SNOOZE_CVE = 'Global Snooze CVE';
export const NODE_CVE_FILTER_APPLIED = 'Node CVE Filter Applied';
export const NODE_CVE_ENTITY_CONTEXT_VIEWED = 'Node CVE Entity Context View';
export const PLATFORM_CVE_FILTER_APPLIED = 'Platform CVE Filter Applied';
export const PLATFORM_CVE_ENTITY_CONTEXT_VIEWED = 'Platform CVE Entity Context View';

// cluster-init-bundles
export const CREATE_INIT_BUNDLE_CLICKED = 'Create Init Bundle Clicked';
export const SECURE_A_CLUSTER_LINK_CLICKED = 'Secure a Cluster Link Clicked';
export const LEGACY_SECURE_A_CLUSTER_LINK_CLICKED = 'Legacy Secure a Cluster Link Clicked';
export const CRS_SECURE_A_CLUSTER_LINK_CLICKED = 'CRS Secure a Cluster Link Clicked';
export const DOWNLOAD_INIT_BUNDLE = 'Download Init Bundle';
export const REVOKE_INIT_BUNDLE = 'Revoke Init Bundle';
export const LEGACY_CLUSTER_DOWNLOAD_YAML = 'Legacy Cluster Download YAML';
export const LEGACY_CLUSTER_DOWNLOAD_HELM_VALUES = 'Legacy Cluster Download Helm Values';

// cluster-registration-secrets
export const CREATE_CLUSTER_REGISTRATION_SECRET_CLICKED =
    'Create Cluster Registration Secret Clicked';
export const DOWNLOAD_CLUSTER_REGISTRATION_SECRET = 'Download Cluster Registration Secret';
export const REVOKE_CLUSTER_REGISTRATION_SECRET = 'Revoke Cluster Registration Secret';

// policy violations

export const FILTERED_WORKFLOW_VIEW_SELECTED = 'Filtered Workflow View Selected';
export const POLICY_VIOLATIONS_FILTER_APPLIED = 'Policy Violations Filter Applied';

// compliance

export const COMPLIANCE_REPORT_DOWNLOAD_GENERATION_TRIGGERED =
    'Compliance Report Download Generation Triggered';
export const COMPLIANCE_REPORT_MANUAL_SEND_TRIGGERED = 'Compliance Report Manual Send Triggered';
export const COMPLIANCE_REPORT_JOBS_TABLE_VIEWED = 'Compliance Report Jobs Table Viewed';
export const COMPLIANCE_REPORT_JOBS_VIEW_TOGGLED = 'Compliance Report Jobs View Toggled';
export const COMPLIANCE_REPORT_JOB_STATUS_FILTERED = 'Compliance Report Job Status Filtered';
export const COMPLIANCE_SCHEDULES_WIZARD_SAVE_CLICKED = 'Compliance Schedules Wizard Save Clicked';
export const COMPLIANCE_SCHEDULES_WIZARD_STEP_CHANGED = 'Compliance Schedules Wizard Step Changed';

/**
 * Boolean fields should be tracked with 0 or 1 instead of true/false. This
 * allows us to use the boolean fields in numeric aggregations in the
 * analytics dashboard to retrieve an accurate count of the number of times
 * a property was enabled for an event.
 */
type AnalyticsBoolean = 0 | 1;

/**
 * A curated list of filters that we would like to track both the filter category and the
 * filter value. This list should exclude anything that could be considered sensitive or
 * specific to a customer environment. This items in this list must also match the casing of
 * the applied filter _exactly_, otherwise it will be tracked without the filter value.
 */
export const searchCategoriesWithFilter = [
    'Component Source',
    'SEVERITY',
    'FIXABLE',
    'CLUSTER CVE FIXABLE',
    'CVSS',
    'Node Top CVSS',
    'Category',
    'Severity',
    'Lifecycle Stage',
    'Resource Type',
    'Inactive Deployment',
    'Control',
    'Compliance Check Name',
    'Compliance State',
    'Cluster Type',
    'Cluster Platform Type',
    'Standard',
    // 'groupBy' is not a real filter, but is used under the 's' key in the URL in old compliance pages
    'groupBy',
] as const;

export const isSearchCategoryWithFilter = tupleTypeGuard(searchCategoriesWithFilter);
export type SearchCategoryWithFilter = UnionFrom<typeof searchCategoriesWithFilter>;

/**
 * An AnalyticsEvent is either a simple string that represents the event name,
 * or an object with an event name and additional properties.
 */
export type AnalyticsEvent =
    | typeof CLUSTER_CREATED
    | typeof INVITE_USERS_MODAL_OPENED
    | typeof INVITE_USERS_SUBMITTED
    /** Tracks each time a cluster level simulator is opened on Network Graph */
    | {
          event: typeof CLUSTER_LEVEL_SIMULATOR_OPENED;
          properties: {
              cluster: number;
              namespaces: number;
              deployments: number;
          };
      }
    /** Tracks each time network policies are generated on Network Graph */
    | {
          event: typeof GENERATE_NETWORK_POLICIES;
          properties: {
              cluster: number;
              namespaces: number;
              deployments: number;
          };
      }
    /** Tracks each time network policies are downloaded on Network Graph */
    | {
          event: typeof DOWNLOAD_NETWORK_POLICIES;
          properties: {
              cluster: number;
              namespaces: number;
              deployments: number;
          };
      }
    /** Tracks each time CIDR Block form opened on Network Graph */
    | {
          event: typeof CIDR_BLOCK_FORM_OPENED;
          properties: {
              cluster: number;
              namespaces: number;
              deployments: number;
          };
      }
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
     * Tracks each time the user applies a filter on a VM page.
     * This is controlled by the main search bar on all VM CVE pages.
     * We only track the value of the applied filter when it does not represent
     * specifics of a customer environment.
     */
    | {
          event:
              | typeof WORKLOAD_CVE_FILTER_APPLIED
              | typeof NODE_CVE_FILTER_APPLIED
              | typeof PLATFORM_CVE_FILTER_APPLIED;
          properties: { category: SearchCategoryWithFilter; filter: string } | { category: string };
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
    | {
          event: typeof WORKLOAD_CVE_DEFERRAL_EXCEPTION_REQUESTED;
          properties:
              | { expiryType: 'CUSTOM_DATE' | 'TIME'; expiryDays: number }
              | { expiryType: 'ALL_CVE_FIXABLE' | 'ANY_CVE_FIXABLE' | 'INDEFINITE' };
      }
    | {
          event: typeof WORKLOAD_CVE_FALSE_POSITIVE_EXCEPTION_REQUESTED;
          properties: Record<string, never>;
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
     * Tracks each time the user generates an SBOM for an image.
     */
    | typeof IMAGE_SBOM_GENERATED
    /**
     * Tracks each time the user snoozes a Node or Platform CVE via
     * Vulnerability Management 1.0
     */
    | {
          event: typeof GLOBAL_SNOOZE_CVE;
          properties: {
              type: 'NODE' | 'PLATFORM';
              duration: string;
          };
      }
    /**
     * Tracks each view of a CVE entity context (CVE or Node). This is
     * controlled by the entity tabs on the Overview page.
     */
    | {
          event: typeof NODE_CVE_ENTITY_CONTEXT_VIEWED;
          properties: {
              type: 'CVE' | 'Node';
              page: 'Overview';
          };
      }
    /**
     * Tracks each view of a CVE entity context (CVE or Cluster). This is
     * controlled by the entity tabs on the Overview page.
     */
    | {
          event: typeof PLATFORM_CVE_ENTITY_CONTEXT_VIEWED;
          properties: {
              type: 'CVE' | 'Cluster';
              page: 'Overview';
          };
      }
    /**
     * Tracks each time the user clicks the "Create Bundle" button
     */
    | {
          event: typeof CREATE_INIT_BUNDLE_CLICKED;
          properties: {
              source: 'No Clusters' | 'Cluster Init Bundles';
          };
      }
    /**
     * Tracks each time the user clicks the "Create Cluster Registration Secrets" button
     */
    | {
          event: typeof CREATE_CLUSTER_REGISTRATION_SECRET_CLICKED;
          properties: {
              source: 'No Clusters' | 'Cluster Registration Secrets';
          };
      }
    /**
     * Tracks each time the user clicks a link to visit the "Secure a Cluster" page
     */
    | {
          event: typeof SECURE_A_CLUSTER_LINK_CLICKED;
          properties: {
              source: 'No Clusters' | 'Secure a Cluster Dropdown';
          };
      }
    /**
     * Tracks each time the user clicks a link to visit the "CRS Secure a Cluster" page
     */
    | {
          event: typeof CRS_SECURE_A_CLUSTER_LINK_CLICKED;
          properties: {
              source: 'No Clusters' | 'Secure a Cluster Dropdown';
          };
      }
    /**
     * Tracks each time the user clicks a link to visit the legacy installation method page
     */
    | {
          event: typeof LEGACY_SECURE_A_CLUSTER_LINK_CLICKED;
          properties: {
              source: 'No Clusters' | 'Secure a Cluster Dropdown';
          };
      }
    /**
     * Tracks each time the user downloads an init bundle
     */
    | typeof DOWNLOAD_INIT_BUNDLE
    /**
     * Tracks each time the user downloads a cluster registration secret
     */
    | typeof DOWNLOAD_CLUSTER_REGISTRATION_SECRET
    /**
     * Tracks each time the user revokes an init bundle
     */
    | typeof REVOKE_INIT_BUNDLE
    /**
     * Tracks each time the user revokes cluster registration secret
     */
    | typeof REVOKE_CLUSTER_REGISTRATION_SECRET
    /**
     * Tracks each time the user downloads a cluster's YAML file and keys
     */
    | typeof LEGACY_CLUSTER_DOWNLOAD_YAML
    /**
     * Tracks each time the user downloads a cluster's Helm values
     */
    | typeof LEGACY_CLUSTER_DOWNLOAD_HELM_VALUES
    /**
     * Tracks each time the user selects a filtered workflow view
     */
    | {
          event: typeof FILTERED_WORKFLOW_VIEW_SELECTED;
          properties: {
              value: 'Application view' | 'Platform view' | 'Full view';
          };
      }
    /**
     * Tracks each time the user applies a filter on the Policy Violations page.
     * We only track the value of the applied filter when it does not represent
     * specifics of a customer environment.
     */
    | {
          event: typeof POLICY_VIOLATIONS_FILTER_APPLIED;
          properties: { category: string; filter: string } | { category: string };
      }
    /**
     * Tracks each time the user generates a compliance report download
     */
    | {
          event: typeof COMPLIANCE_REPORT_DOWNLOAD_GENERATION_TRIGGERED;
          properties: {
              source: 'Table row' | 'Details page';
          };
      }
    /**
     * Tracks each time the user sends a compliance report manually
     */
    | {
          event: typeof COMPLIANCE_REPORT_MANUAL_SEND_TRIGGERED;
          properties: {
              source: 'Table row' | 'Details page';
          };
      }
    /**
     * Tracks each time the user views the compliance report jobs table
     */
    | typeof COMPLIANCE_REPORT_JOBS_TABLE_VIEWED
    /**
     * Tracks each time the user clicks the "View only my jobs" toggle
     */
    | {
          event: typeof COMPLIANCE_REPORT_JOBS_VIEW_TOGGLED;
          properties: {
              view: 'My jobs';
              state: true | false;
          };
      }
    /**
     * Tracks each time the user filters by report run state
     */
    | {
          event: typeof COMPLIANCE_REPORT_JOB_STATUS_FILTERED;
          properties: {
              value: (
                  | 'WAITING'
                  | 'PREPARING'
                  | 'DOWNLOAD_GENERATED'
                  | 'EMAIL_DELIVERED'
                  | 'ERROR'
                  | 'PARTIAL_ERROR'
              )[];
          };
      }
    | {
          event: typeof COMPLIANCE_SCHEDULES_WIZARD_SAVE_CLICKED;
          properties: {
              success: true | false;
              errorMessage: string;
          };
      }
    | {
          event: typeof COMPLIANCE_SCHEDULES_WIZARD_STEP_CHANGED;
          properties: {
              step: string;
          };
      };

export const redactedHostReplacement = 'redacted.host.invalid';
export const redactedSearchReplacement = '*****';

// Replace the hostname, port, and search parameters with redacted values
function redactURL(location: string): string {
    try {
        const url = new URL(location);
        url.host = redactedHostReplacement;
        url.search = redactSearchParams(location);
        return url.toString();
    } catch (error) {
        Raven.captureException(error);
        // Do not throw an error during an analytics event. If an error occurs, redact the entire URL.
        return '';
    }
}

type RawQueryStringValue = qs.ParsedQs[keyof qs.ParsedQs];

function isAllowedSearchKey(key: string): boolean {
    return searchCategoriesWithFilter.some((term) => key.toUpperCase() === term.toUpperCase());
}

// Given a parsed query string value, redact any properties that are not explicitly allowed
function redactParsedQs(value: RawQueryStringValue, key: string): RawQueryStringValue {
    if (typeof value === 'undefined') {
        return value;
    }
    if (Array.isArray(value)) {
        return (
            value
                .map((v: string | qs.ParsedQs) => redactParsedQs(v, key))
                // Our search structure does not allow nested objects, so we can safely filter out any non-string values
                // for simplicity
                .filter((v): v is string => typeof v === 'string')
        );
    }
    if (typeof value === 'object') {
        return mapValues(value, redactParsedQs);
    }
    // The terminal case: if the value is a string, redact it if the key is not allowed
    if (typeof value === 'string') {
        return isAllowedSearchKey(key) ? value : redactedSearchReplacement;
    }

    return ensureExhaustive(value);
}

// Traverse all defined search keys and redact any properties that are not explicitly allowed
// in analytics events.
function redactSearchParams(location: string): string {
    // Top level URL parameters that can contain user or installation-specific information. Any
    // key that should have its value redacted should be added to this list.
    const topLevelSearchKeys = ['s', 's2'];

    try {
        const url = new URL(location);
        const queryObject = getQueryObject(url.search);
        const redactedQueryObject = mapValues(queryObject, (v, k) =>
            topLevelSearchKeys.includes(k) ? redactParsedQs(v, k) : v
        );
        url.search = getQueryString(redactedQueryObject);
        // Re-convert via URL constructor to ensure URI encoding for search keys that matches how Segment would handle this natively
        return new URL(url.toString()).search;
    } catch (error) {
        Raven.captureException(error);
        // Do not throw an error during an analytics event. If an error occurs, redact the entire search string.
        return '';
    }
}

// Strip out installation-specific information from the analytics context
export function getRedactedOriginProperties(location: string) {
    return {
        url: redactURL(location),
        search: redactSearchParams(location),
        // Referrer is unused, so we remove it entirely here to avoid sending private values to analytics
        referrer: '',
    };
}

const useAnalytics = () => {
    const telemetry = useSelector(selectors.publicConfigTelemetrySelector);
    const { enabled: isTelemetryEnabled } = telemetry || ({} as Telemetry);

    const analyticsPageVisit = useCallback(
        (type: string, name: string, additionalProperties = {}): void => {
            if (isTelemetryEnabled !== false) {
                window.analytics?.page(type, name, {
                    ...additionalProperties,
                    ...getRedactedOriginProperties(window.location.toString()),
                });
            }
        },
        [isTelemetryEnabled]
    );

    const analyticsTrack = useCallback(
        (analyticsEvent: AnalyticsEvent): void => {
            if (isTelemetryEnabled === false) {
                return;
            }

            const redactedEventContext = {
                context: {
                    page: getRedactedOriginProperties(window.location.toString()),
                },
            };

            if (typeof analyticsEvent === 'string') {
                window.analytics?.track(analyticsEvent, undefined, redactedEventContext);
            } else {
                window.analytics?.track(
                    analyticsEvent.event,
                    analyticsEvent.properties,
                    redactedEventContext
                );
            }
        },
        [isTelemetryEnabled]
    );

    return { analyticsPageVisit, analyticsTrack };
};

export default useAnalytics;
