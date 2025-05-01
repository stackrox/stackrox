import * as yup from 'yup';

import { VulnerabilitySeverity } from 'types/cve.proto';

export const vulnerabilitySeverityLabels = [
    'Critical',
    'Important',
    'Moderate',
    'Low',
    'Unknown',
] as const;
export type VulnerabilitySeverityLabel = (typeof vulnerabilitySeverityLabels)[number];
export function isVulnerabilitySeverityLabel(value: unknown): value is VulnerabilitySeverityLabel {
    return vulnerabilitySeverityLabels.some((severity) => severity === value);
}

const fixableStatuses = ['Fixable', 'Not fixable'] as const;
export type FixableStatus = (typeof fixableStatuses)[number];
export function isFixableStatus(value: unknown): value is FixableStatus {
    return fixableStatuses.some((status) => status === value);
}

// `QuerySearchFilter` is a restricted subset of the `SearchFilter` obtained from the URL that
// has been parsed to convert values to the format expected by the backend. It also restricts
// the filter values to a `string[]` for consistency.
export type QuerySearchFilter = Partial<
    {
        SEVERITY: VulnerabilitySeverity[];
        FIXABLE: ('true' | 'false')[];
    } & Record<string, string[]>
>;

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
    } catch {
        return false;
    }
}

export const detailsTabValues = ['Vulnerabilities', 'Details', 'Resources'] as const;

export type DetailsTab = (typeof detailsTabValues)[number];

export function isDetailsTab(value: unknown): value is DetailsTab {
    return detailsTabValues.some((tab) => tab === value);
}

export const workloadEntityTabValues = ['CVE', 'Image', 'Deployment'] as const;

export type WorkloadEntityTab = (typeof workloadEntityTabValues)[number];

export const nodeEntityTabValues = ['CVE', 'Node'] as const;

export type NodeEntityTab = (typeof nodeEntityTabValues)[number];

export const platformEntityTabValues = ['CVE', 'Cluster'] as const;

export type PlatformEntityTab = (typeof platformEntityTabValues)[number];

export type EntityTab = WorkloadEntityTab | NodeEntityTab | PlatformEntityTab;

export type WatchStatus = 'WATCHED' | 'NOT_WATCHED' | 'UNKNOWN';

export type CveExceptionRequestType = 'DEFERRAL' | 'FALSE_POSITIVE';

export const observedCveModeValues = ['WITH_CVES', 'WITHOUT_CVES'] as const;

export type ObservedCveMode = (typeof observedCveModeValues)[number];

export function isObservedCveMode(value: unknown): value is ObservedCveMode {
    return observedCveModeValues.some((mode) => mode === value);
}

export type VerifiedStatus =
    | 'CORRUPTED_SIGNATURE'
    | 'FAILED_VERIFICATION'
    | 'GENERIC_ERROR'
    | 'INVALID_SIGNATURE_ALGO'
    | 'UNSET'
    | 'VERIFIED';

export type SignatureVerificationResult = {
    status: VerifiedStatus;
    verificationTime: string; // ISO 8601 formatted date time.
    verifiedImageReferences: string[];
    verifierId: string; // Signature integration id of the form `io.stackrox.signatureintegration.<uuid>`.
};
