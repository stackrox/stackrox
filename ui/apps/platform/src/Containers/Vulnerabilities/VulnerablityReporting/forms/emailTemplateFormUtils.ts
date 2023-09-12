import * as yup from 'yup';
import { FormikProps } from 'formik';

import {
    ReportFormValues,
    emailBodyValidation,
    emailSubjectValidation,
} from './useReportFormValues';

// Helper functions

export const defaultEmailSubjectTemplate =
    'RHACS Workload CVE Report for <Config name>; Scope: <Collection name>';

export const defaultEmailBody =
    'Red Hat Advanced Cluster Security (RHACS) for Kubernetes has identified workload CVEs in the images matched by the following report configuration parameters. The attached Vulnerability report lists those workload CVEs and associated details to help with remediation. Please review the vulnerable software packages/components from the impacted images and update them to a version containing the fix, if one is available.\n\nPlease note that an attachment of the report data will not be provided if no CVEs are found.';

export const defaultEmailBodyWithNoCVEsFound =
    'Red Hat Advanced Cluster Security (RHACS) for Kubernetes has identified no workload CVEs in the images matched by the following report configuration parameters.\n\nPlease note that an attachment of the report data will not be provided if no CVEs are found.';

export function getDefaultEmailSubject(formik: FormikProps<ReportFormValues>): string {
    return defaultEmailSubjectTemplate
        .replace('<Config name>', formik.values.reportParameters.reportName)
        .replace('<Collection name>', formik.values.reportParameters.reportScope?.name || '');
}

export function isDefaultEmailSubject(emailSubject: string): boolean {
    // Create a regex from the template by escaping special characters and replacing the markers with regex wildcards
    const regex = new RegExp(
        `^${defaultEmailSubjectTemplate
            .replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // escape special chars
            .replace(/<Config name>/g, '.*') // replace <Config name> marker
            .replace(/<Collection name>/g, '.*')}$` // replace <Collection name> marker
    );

    return regex.test(emailSubject);
}

export function isDefaultEmailTemplate(emailSubject: string, emailBody: string): boolean {
    return emailSubject === '' && emailBody === '';
}

// Validation

export const emailTemplateValidationSchema = yup.object({
    emailSubject: emailSubjectValidation,
    emailBody: emailBodyValidation,
});

export type EmailTemplateFormData = yup.InferType<typeof emailTemplateValidationSchema>;
