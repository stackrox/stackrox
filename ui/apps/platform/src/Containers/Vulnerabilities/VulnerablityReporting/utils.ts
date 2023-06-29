import { VulnerabilitySeverity } from 'types/cve.proto';
import { ImageType } from 'types/reportConfigurationService.proto';
import { CVESDiscoveredSince, CVEStatus } from './forms/useReportFormValues';

export const cveSeverityLabelMap: Record<VulnerabilitySeverity, string> = {
    UNKNOWN_VULNERABILITY_SEVERITY: 'Unknown',
    CRITICAL_VULNERABILITY_SEVERITY: 'Critical',
    IMPORTANT_VULNERABILITY_SEVERITY: 'Important',
    MODERATE_VULNERABILITY_SEVERITY: 'Moderate',
    LOW_VULNERABILITY_SEVERITY: 'Low',
};

export const cveStatusLabelMap: Record<CVEStatus, string> = {
    FIXABLE: 'Fixable',
    NOT_FIXABLE: 'Not fixable',
};

export const imageTypeLabelMap: Record<ImageType, string> = {
    DEPLOYED: 'Deployed images',
    WATCHED: 'Watched images',
};

export const cvesDiscoveredSinceLabelMap: Record<CVESDiscoveredSince, string> = {
    ALL_VULN: 'All time',
    SINCE_LAST_REPORT: 'Last successful scheduled run report',
    START_DATE: 'Custom start date',
};
