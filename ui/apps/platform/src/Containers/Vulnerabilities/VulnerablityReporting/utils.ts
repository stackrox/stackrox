import {
    Fixability,
    ImageType,
    ReportConfiguration,
    ReportStatus,
    Schedule,
    VulnerabilityReportFilters,
    VulnerabilityReportFiltersBase,
} from 'services/ReportsService.types';
import { DayOfMonth, DayOfWeek } from 'Components/PatternFly/DayPickerDropdown';
import {
    CVESDiscoveredSince,
    CVESDiscoveredStartDate,
    DeliveryDestination,
    ReportFormValues,
} from './forms/useReportFormValues';

export const imageTypeLabelMap: Record<ImageType, string> = {
    DEPLOYED: 'Deployed images',
    WATCHED: 'Watched images',
};

export const cvesDiscoveredSinceLabelMap: Record<CVESDiscoveredSince, string> = {
    ALL_VULN: 'All time',
    SINCE_LAST_REPORT: 'Last successful scheduled run report',
    START_DATE: 'Custom start date',
};

export const commaSeparateWithAnd = (arr: string[]) => {
    if (arr.length === 0) {
        return '';
    }
    if (arr.length === 1) {
        return arr[0];
    }
    const last = arr.pop();
    if (!last) {
        return arr.join(', ');
    }
    return `${arr.join(', ')} and ${last}`;
};

/*
 * This function will take the report form values from the forms for the create, edit, and clone views
 * and transform them into the report configuration object to be sent through the APIs
 */
export function getReportConfigurationFromFormValues(
    formValues: ReportFormValues
): ReportConfiguration {
    const { reportId, reportParameters, deliveryDestinations, schedule: formSchedule } = formValues;

    // transform form values to values to be sent through API
    const fixability: Fixability =
        reportParameters.cveStatus.length > 1 ? 'BOTH' : reportParameters.cveStatus[0];

    const vulnReportFiltersBase: VulnerabilityReportFiltersBase = {
        fixability,
        severities: reportParameters.cveSeverities,
        imageTypes: reportParameters.imageType,
    };
    let vulnReportFilters: VulnerabilityReportFilters;
    if (reportParameters.cvesDiscoveredSince === 'SINCE_LAST_REPORT') {
        vulnReportFilters = {
            ...vulnReportFiltersBase,
            sinceLastSentScheduledReport: true,
        };
    } else if (
        reportParameters.cvesDiscoveredSince === 'START_DATE' &&
        reportParameters.cvesDiscoveredStartDate
    ) {
        vulnReportFilters = {
            ...vulnReportFiltersBase,
            sinceStartDate: new Date(reportParameters.cvesDiscoveredStartDate).toISOString(),
        };
    } else {
        vulnReportFilters = {
            ...vulnReportFiltersBase,
            allVuln: true,
        };
    }

    const notifiers = deliveryDestinations.map((deliveryDestination) => {
        return {
            emailConfig: {
                notifierId: deliveryDestination.notifier?.id || '',
                mailingLists: deliveryDestination.mailingLists,
            },
            notifierName: deliveryDestination.notifier?.name || '',
        };
    });

    let schedule: Schedule | null;
    if (formSchedule.intervalType === 'WEEKLY') {
        schedule = {
            intervalType: 'WEEKLY',
            hour: 0,
            minute: 0,
            daysOfWeek: {
                days: formSchedule.daysOfWeek?.map((day) => Number(day)) || [],
            },
        };
    } else if (formSchedule.intervalType === 'MONTHLY') {
        schedule = {
            intervalType: 'MONTHLY',
            hour: 0,
            minute: 0,
            daysOfMonth: {
                days: formSchedule.daysOfMonth?.map((day) => Number(day)) || [],
            },
        };
    } else {
        schedule = null;
    }

    const reportConfiguration: ReportConfiguration = {
        id: reportId,
        name: reportParameters.reportName,
        description: reportParameters.description,
        type: 'VULNERABILITY',
        vulnReportFilters,
        resourceScope: {
            collectionScope: {
                collectionId: reportParameters.reportScope?.id || '',
                collectionName: reportParameters.reportScope?.name || '',
            },
        },
        notifiers,
        schedule,
    };

    return reportConfiguration;
}

/*
 * This function will take the report configuration object and transform it into the report form
 * values to be used in the forms for the create, edit, and clone views
 */
export function getReportFormValuesFromConfiguration(
    reportConfiguration: ReportConfiguration
): ReportFormValues {
    const { id, name, description, vulnReportFilters, resourceScope, notifiers, schedule } =
        reportConfiguration;

    let cvesDiscoveredSince: CVESDiscoveredSince = 'ALL_VULN';
    let cvesDiscoveredStartDate: CVESDiscoveredStartDate;

    if ('allVuln' in vulnReportFilters) {
        cvesDiscoveredSince = 'ALL_VULN';
    } else if ('sinceLastSentScheduledReport' in vulnReportFilters) {
        cvesDiscoveredSince = 'SINCE_LAST_REPORT';
    } else if ('sinceStartDate' in vulnReportFilters) {
        cvesDiscoveredSince = 'START_DATE';
        // Strip off the google.protobuf.Timestamp time portion of the date string
        cvesDiscoveredStartDate = vulnReportFilters.sinceStartDate.substring(0, 10);
    } else {
        // we'll default to this if none of these fields are present
        cvesDiscoveredSince = 'ALL_VULN';
    }

    const deliveryDestinations = notifiers.map((notifier) => {
        const deliveryDestination: DeliveryDestination = {
            notifier: {
                id: notifier.emailConfig.notifierId,
                name: notifier.notifierName,
            },
            mailingLists: notifier.emailConfig.mailingLists,
        };
        return deliveryDestination;
    });

    let formSchedule: ReportFormValues['schedule'];
    if (!schedule) {
        formSchedule = {
            intervalType: null,
            daysOfWeek: [],
            daysOfMonth: [],
        };
    } else if (schedule.intervalType === 'WEEKLY') {
        formSchedule = {
            intervalType: 'WEEKLY',
            daysOfWeek: schedule.daysOfWeek.days.map((day) => String(day) as DayOfWeek),
            daysOfMonth: [],
        };
    } else {
        formSchedule = {
            intervalType: 'MONTHLY',
            daysOfWeek: [],
            daysOfMonth: schedule.daysOfMonth.days.map((day) => String(day) as DayOfMonth),
        };
    }

    const reportFormValues: ReportFormValues = {
        reportId: id,
        reportParameters: {
            reportName: name,
            description,
            cveSeverities: vulnReportFilters.severities,
            cveStatus:
                vulnReportFilters.fixability === 'BOTH'
                    ? ['FIXABLE', 'NOT_FIXABLE']
                    : [vulnReportFilters.fixability],
            imageType: vulnReportFilters.imageTypes,
            cvesDiscoveredSince,
            cvesDiscoveredStartDate,
            reportScope: {
                id: resourceScope.collectionScope.collectionId,
                name: resourceScope.collectionScope.collectionName,
            },
        },
        deliveryDestinations,
        schedule: formSchedule,
    };

    return reportFormValues;
}

export function getReportStatusText(reportStatus: ReportStatus | null): string {
    let statusText = '-';

    if (
        reportStatus?.runState === 'SUCCESS' &&
        reportStatus?.reportNotificationMethod === 'EMAIL'
    ) {
        statusText = 'Emailed';
    } else if (
        reportStatus?.runState === 'SUCCESS' &&
        reportStatus?.reportNotificationMethod === 'DOWNLOAD'
    ) {
        statusText = 'Download prepared';
    } else if (
        reportStatus?.runState === 'FAILURE' &&
        reportStatus?.reportNotificationMethod === 'EMAIL'
    ) {
        statusText = 'Email attempted';
    } else if (
        reportStatus?.runState === 'FAILURE' &&
        reportStatus?.reportNotificationMethod === 'DOWNLOAD'
    ) {
        statusText = 'Failed to generate download';
    } else if (reportStatus?.runState === 'SUCCESS') {
        statusText = 'Success';
    } else if (reportStatus?.runState === 'FAILURE') {
        statusText = 'Error';
    } else if (reportStatus?.runState === 'PREPARING') {
        statusText = 'Preparing';
    } else if (reportStatus?.runState === 'WAITING') {
        statusText = 'Waiting';
    }

    return statusText;
}
