import { Policy } from 'types/policy.proto';
import { ImportPolicyResponse } from 'services/PoliciesService';

export const MIN_POLICY_NAME_LENGTH = 5;

export type PolicyImportErrorDuplicateName = {
    type: 'duplicate_name';
    duplicateName: string;
} & PolicyImportErrorBase;

export type PolicyImportErrorDuplicateId = {
    type: 'duplicate_id';
    duplicateName: string;
    incomingName: string;
    incomingId: string;
} & PolicyImportErrorBase;

export type PolicyImportErrorInvalidPolicy = {
    type: 'invalid_policy';
    validationError: string;
} & PolicyImportErrorBase;

type PolicyImportErrorBase = {
    type: string;
    message: string;
};

export type PolicyImportError =
    | PolicyImportErrorDuplicateName
    | PolicyImportErrorDuplicateId
    | PolicyImportErrorInvalidPolicy;

/**
 * parsePolicyImportErrors extracts any errors from the array of policies in the import, for ease-of-use in the UI
 */
export function parsePolicyImportErrors(responses: ImportPolicyResponse[]): PolicyImportError[][] {
    const errors = responses.reduce((acc: PolicyImportError[][], res) => {
        if (res.errors?.length) {
            const errorItems = res.errors.reduce((errList: PolicyImportError[], err) => {
                if (err.type === 'duplicate_id') {
                    errList.push({
                        type: 'duplicate_id',
                        duplicateName: err.duplicateName,
                        incomingName: res.policy?.name,
                        incomingId: res.policy?.id,
                        message: err.message,
                    });
                } else if (err.type === 'duplicate_name') {
                    errList.push({
                        type: 'duplicate_name',
                        duplicateName: err.duplicateName,
                        message: err.message,
                    });
                } else if (err.type === 'invalid_policy') {
                    errList.push({
                        type: 'invalid_policy',
                        validationError: err.validationError,
                        message: err.message,
                    });
                }
                return errList;
            }, []);

            return acc.concat([errorItems]);
        }
        return [...acc];
    }, []);

    return errors;
}

export type PolicyResolutionType = 'rename' | 'overwrite' | 'keepBoth';

export type PolicyResolution = {
    resolution: PolicyResolutionType | null;
    newName: string;
};

/**
 * isDuplicateResolved performs a check of the object for a Duplicate Policy Form,
 *   and determines if user has chosen a combination of inputs that will resolve
 *   the duplication if the policy is re-submitted
 */
export function isDuplicateResolved(resolutionObj: PolicyResolution): boolean {
    return (
        resolutionObj.resolution === 'overwrite' ||
        resolutionObj.resolution === 'keepBoth' ||
        (resolutionObj.resolution === 'rename' &&
            resolutionObj?.newName?.length >= MIN_POLICY_NAME_LENGTH)
    );
}

type PolicyErrorMessage = {
    type: string;
    msg: string;
};

/**
 * stringify any import errors to display to the user
 */
export function getErrorMessages(policyErrors: PolicyImportError[]): PolicyErrorMessage[] {
    const errorMessages = policyErrors.map((err) => {
        let msg = '';
        switch (err.type) {
            case 'duplicate_id': {
                msg = `An existing policy with the name “${err.duplicateName}” has the same ID—${err.incomingId}—as the policy “${err.incomingName}” you are trying to import.`;
                break;
            }
            case 'duplicate_name': {
                msg = `An existing policy has the same name, “${err.duplicateName}”, as the one you are trying to import.`;
                break;
            }
            case 'invalid_policy':
            default: {
                msg = `${err.message}: ${err.validationError}`;
                break;
            }
        }

        return {
            type: err.type,
            msg,
        };
    });

    return errorMessages;
}

/**
 * modify the import payload to reflect the duplicate resolution chosen by the user
 */
export function getResolvedPolicies(
    policies: Policy[],
    errors: PolicyImportError[],
    duplicateResolution: PolicyResolution
): [Policy[], { overwrite?: boolean }] {
    const resolvedPolicies = [...policies];
    const metadata = {
        overwrite: false,
    };

    if (errors) {
        if (duplicateResolution?.resolution === 'overwrite') {
            metadata.overwrite = true;
        } else if (duplicateResolution?.resolution === 'keepBoth') {
            resolvedPolicies[0].id = '';
        } else if (duplicateResolution?.resolution === 'rename') {
            resolvedPolicies[0].name = duplicateResolution?.newName;

            if (errors.some((err) => err.type === 'duplicate_id')) {
                resolvedPolicies[0].id = '';
            }
        }
    }

    return [resolvedPolicies, metadata];
}

/**
 * simple function to abstract the test for only a duplicate ID error from the backend
 *
 * @param   {array}  importErrors  Array< type: string, value: string } >
 *
 * @return  {boolean}              true if the only error is a duplicate policy ID
 */
export function hasDuplicateIdOnly(importErrors: PolicyImportError[]): boolean {
    return importErrors?.length === 1 && importErrors[0].type === 'duplicate_id';
}

/**
 * simple function to abstract the test for only duplicate errors
 */
export function checkDupeOnlyErrors(importErrors: PolicyImportError[][]): boolean {
    return !!(
        importErrors?.length &&
        importErrors.every((policyErrors) => {
            const hasInvalidPolicy = policyErrors.some((err) => err.type === 'invalid_policy');
            const hasDupeErrors = policyErrors.some((err) => err.type.includes('dup'));

            return hasDupeErrors && !hasInvalidPolicy;
        })
    );
}

/**
 * function to abstract checks for whether importing is currently blocked
 */
export function checkForBlockedSubmit({
    numPolicies,
    messageType,
    hasDuplicateErrors,
    duplicateResolution,
}: {
    numPolicies: number;
    messageType: string;
    hasDuplicateErrors: boolean;
    duplicateResolution: PolicyResolution;
}): boolean {
    return (
        numPolicies < 1 || // at least one policy must be in selected file
        messageType === 'info' || // an info message means upload has already succeeded
        (!hasDuplicateErrors && messageType === 'error') || // error without dupes means a validation error
        (hasDuplicateErrors && !isDuplicateResolved(duplicateResolution)) // dupes, not resolved by user yet
    );
}
