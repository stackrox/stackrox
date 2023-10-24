import { addDays, isAfter } from 'date-fns';
import * as yup from 'yup';

import {
    CreateDeferVulnerabilityExceptionRequest,
    CreateFalsePositiveVulnerabilityExceptionRequest,
} from 'services/VulnerabilityExceptionService';
import { ensureExhaustive } from 'utils/type.utils';

export type ScopeContext = 'GLOBAL' | { image: { name: string; tag: string } };

export const exceptionValidationSchema = yup.object({
    cves: yup.array().of(yup.string()).min(1, 'At least one CVE must be selected'),
    comment: yup.string().required('A rationale is required'),
});

export type ExceptionValues = {
    cves: string[];
    comment: string;
    scope: {
        imageScope: {
            registry: string;
            remote: string;
            tag: string;
        };
    };
};

export type FalsePositiveValues = ExceptionValues;

export type DeferralValues = ExceptionValues & {
    expiry?:
        | { type: 'TIME'; days: number }
        | { type: 'ALL_CVE_FIXABLE' | 'ANY_CVE_FIXABLE' | 'INDEFINITE' }
        | { type: 'CUSTOM_DATE'; date: string };
};

/**
 * Validates that a date is in the future
 * @param date
 * @returns An error message if the date is in the past, otherwise an empty string
 */
export function futureDateValidator(date: Date): string {
    return isAfter(new Date(), date) ? 'Date must be in the future' : '';
}

/**
 * Converts form values to a request body for creating a deferral exception. The `expiry` field
 * has been separated from the rest of the form values to ensure that it is not null. Null checking
 * is done at the caller level.
 *
 * @param formValues
 * @param expiry
 * @returns A request body for creating a deferral exception
 */
export function formValuesToDeferralRequest(
    formValues: Omit<DeferralValues, 'expiry'>,
    expiry: Required<DeferralValues>['expiry']
): CreateDeferVulnerabilityExceptionRequest {
    function requestWithExpiry(
        exceptionExpiry: CreateDeferVulnerabilityExceptionRequest['exceptionExpiry']
    ) {
        return {
            cves: formValues.cves,
            comment: formValues.comment,
            scope: formValues.scope,
            exceptionExpiry,
        };
    }

    const expiryType = expiry.type;

    switch (expiryType) {
        case 'ALL_CVE_FIXABLE':
        case 'ANY_CVE_FIXABLE':
            return requestWithExpiry({ expiryType });
        case 'TIME': {
            const expiresOn = addDays(Date.now(), expiry.days).toISOString();
            return requestWithExpiry({ expiryType, expiresOn });
        }
        case 'CUSTOM_DATE':
            return requestWithExpiry({ expiryType: 'TIME', expiresOn: expiry.date });
        case 'INDEFINITE':
            return requestWithExpiry({ expiryType: 'TIME', expiresOn: null });
        default:
            return ensureExhaustive(expiryType);
    }
}

export function formValuesToFalsePositiveRequest(
    formValues: FalsePositiveValues
): CreateFalsePositiveVulnerabilityExceptionRequest {
    return {
        cves: formValues.cves,
        comment: formValues.comment,
        scope: formValues.scope,
    };
}
