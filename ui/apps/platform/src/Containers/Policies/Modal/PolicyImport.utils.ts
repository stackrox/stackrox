import { Policy } from 'types/policy.proto';

export const MIN_POLICY_NAME_LENGTH = 5;

type PolicyImportError = {
    type: string;
    duplicateName: string;
    validationError?: string;
    message: string;
}

type PolicyImportResponse = { 
    succeeded: boolean; 
    policy?: {
        name: string;
        id: string;
    }; 
    errors?: PolicyImportError[];
}

type PolicyImportErrorListItem = {
    type: string;
    incomingName?: string;
    incomingId?: string;
    duplicateName: string;
    validationError?: string | null;
    message: string;
}

/**
 * parsePolicyImportErrors extracts any errors from the array of policies in the import, for ease-of-use in the UI
 */
export function parsePolicyImportErrors(responses: PolicyImportResponse[]): PolicyImportErrorListItem[][] {
    const errors = responses.reduce((acc: PolicyImportErrorListItem[][], res) => {
        if (res.errors?.length) {
            const errorItems = res.errors.reduce((errList: PolicyImportErrorListItem[], err: PolicyImportError) => {
                const thisErr = {
                    type: err.type,
                    incomingName: res.policy?.name,
                    incomingId: res.policy?.id,
                    duplicateName: err.duplicateName,
                    validationError: err?.validationError || null,
                    message: err.message,
                };

                return errList.concat(thisErr);
            }, []);

            return acc.concat([errorItems]);
        }
        return [...acc];
    }, []);

    return errors;
}

type PolicyResolution = {
    resolution: 'rename'| 'overwrite' | 'keepBoth';
    newName: string;
}

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
}

/**
 * stringify any import errors to display to the user
 */
export function getErrorMessages(policyErrors: PolicyImportErrorListItem[]): PolicyErrorMessage[] {
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
export function getResolvedPolicies(policies: Policy[], errors: PolicyImportErrorListItem[], duplicateResolution: PolicyResolution): [Policy[], { overwrite?: boolean }] {
    const resolvedPolicies = [...policies];
    const metadata = {
        overwrite: false
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
export function hasDuplicateIdOnly(importErrors: PolicyImportErrorListItem[]): boolean {
    return importErrors?.length === 1 && importErrors[0].type === 'duplicate_id';
}

/**
 * simple function to abstract the test for only duplicate errors
 */
export function checkDupeOnlyErrors(importErrors: PolicyImportErrorListItem[][]): boolean {
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
