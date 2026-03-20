import type { ComplianceProfileOperatorKind } from 'services/ComplianceCommon';

export const DEFAULT_COMPLIANCE_PAGE_SIZE = 10;

// searchable and sortable query parameter fields
export const SCAN_CONFIG_NAME_QUERY = 'Compliance Scan Config Name';

// Display labels for backend enums
export const complianceProfileOperatorKindLabels: Record<ComplianceProfileOperatorKind, string> =
    Object.freeze({
        OPERATOR_KIND_UNSPECIFIED: 'Unspecified',
        PROFILE: 'Profile',
        TAILORED_PROFILE: 'Tailored',
    });
