import { VulnerabilitySeverity } from 'types/cve.proto';
import * as yup from 'yup';

const vulnerabilitySeverityLabels = ['Critical', 'Important', 'Moderate', 'Low'] as const;
export type VulnerabilitySeverityLabel = (typeof vulnerabilitySeverityLabels)[number];
export function isVulnerabilitySeverityLabel(value: unknown): value is VulnerabilitySeverityLabel {
    return vulnerabilitySeverityLabels.some((severity) => severity === value);
}

const fixableStatuses = ['Fixable', 'Not fixable'] as const;
export type FixableStatus = (typeof fixableStatuses)[number];
export function isFixableStatus(value: unknown): value is FixableStatus {
    return fixableStatuses.some((status) => status === value);
}

// `QuerySearchFilter` is a restricted subset of the `SearchFilter` obtained from the URL that only
// supports search keys that are valid in the Workload CVE section of the app
export type QuerySearchFilter = Partial<{
    SEVERITY: VulnerabilitySeverity[];
    FIXABLE: ('true' | 'false')[];
    CVE: string[];
    IMAGE: string[];
    DEPLOYMENT: string[];
    NAMESPACE: string[];
    CLUSTER: string[];
}>;

const vulnMgmtLocalStorageSchema = yup.object({
    preferences: yup.object({
        defaultFilters: yup.object({
            SEVERITY: yup
                .array(yup.string().required().oneOf(vulnerabilitySeverityLabels))
                .required(),
            FIXABLE: yup.array(yup.string().required().oneOf(fixableStatuses)).required(),
        }),
    }),
});

export type VulnMgmtLocalStorage = yup.InferType<typeof vulnMgmtLocalStorageSchema>;

export type DefaultFilters = VulnMgmtLocalStorage['preferences']['defaultFilters'];

export function isVulnMgmtLocalStorage(value: unknown): value is VulnMgmtLocalStorage {
    try {
        vulnMgmtLocalStorageSchema.validateSync(value);
        return true;
    } catch (error) {
        return false;
    }
}

export const detailsTabValues = ['Vulnerabilities', 'Resources'] as const;

export type DetailsTab = (typeof detailsTabValues)[number];

export function isDetailsTab(value: unknown): value is DetailsTab {
    return detailsTabValues.some((tab) => tab === value);
}

export const entityTabValues = ['CVE', 'Image', 'Deployment'] as const;

export type EntityTab = (typeof entityTabValues)[number];

export function isValidEntityTab(value: unknown): value is EntityTab {
    return entityTabValues.some((tab) => tab === value);
}

export type WatchStatus = 'WATCHED' | 'NOT_WATCHED' | 'UNKNOWN';

export type CveExceptionRequestType = 'DEFERRAL' | 'FALSE_POSITIVE';
