export const MIN_POLICY_NAME_LENGTH = 5;

export const POLICY_DUPE_ACTIONS = {
    KEEP_BOTH: 'keepBoth',
    RENAME: 'rename',
    OVERWRITE: 'overwrite',
};

/**
 * parsePolicyImportErrors extracts any errors from the array of policies in the import, for ease-of-use in the UI
 *
 * @param   {array}  responses  a list of objects { succeeded: boolean, policy: object, errors: Array<object> }
 *
 * @return  {[array]}           Array< Array<{ type: string, incomingName: string, incomingId: string. duplicateName: string } > >
 */
export function parsePolicyImportErrors(responses = []) {
    const errors = responses.reduce((acc, res) => {
        if (res?.errors?.length) {
            const errorItems = res.errors.reduce((errList, err) => {
                const thisErr = {
                    type: err.type,
                    incomingName: res?.policy?.name,
                    incomingId: res?.policy?.id,
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

/**
 * isDuplicateResolved performs a check of the object for a Duplicate Policy Form,
 *   and determines if user has chosen a combination of inputs that will resolve
 *   the duplication if the policy is re-submitted
 *
 * @param   {object}  resolutionObj  { resolution: oneOf(POLICY_DUPE_ACTIONS.RENAME|POLICY_DUPE_ACTIONS.OVERWRITE), newName: string }
 *
 * @return  {boolean}                 true if policy can be re-submitted, false otherwise
 */
export function isDuplicateResolved(resolutionObj) {
    return (
        resolutionObj.resolution === POLICY_DUPE_ACTIONS.OVERWRITE ||
        resolutionObj.resolution === POLICY_DUPE_ACTIONS.KEEP_BOTH ||
        (resolutionObj.resolution === POLICY_DUPE_ACTIONS.RENAME &&
            resolutionObj?.newName?.length >= MIN_POLICY_NAME_LENGTH)
    );
}

/**
 * stringify any import errors to display to the user
 *
 * @param   {array}  policyErrors  Array< { type: string, value: string } >
 *
 * @return  {array}               each array and value, joined by "and"
 */
export function getErrorMessages(policyErrors) {
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
 *
 * @param   {array}  policies             Array< policy{object} >
 * @param   {array}  errors               Array < type: string, value: string } >
 * @param   {object} duplicateResolution  < resolution: string, newName: string } >
 *
 * @return  {tuple}                       First element: Array< object[policy], second element: metadata{ overwrite?: boolean }
 */
export function getResolvedPolicies(policies, errors, duplicateResolution) {
    const resolvedPolicies = [...policies];
    const metadata = {};

    if (errors) {
        if (duplicateResolution?.resolution === POLICY_DUPE_ACTIONS.OVERWRITE) {
            metadata.overwrite = true;
        } else if (duplicateResolution?.resolution === POLICY_DUPE_ACTIONS.KEEP_BOTH) {
            resolvedPolicies[0].id = '';
        } else if (duplicateResolution?.resolution === POLICY_DUPE_ACTIONS.RENAME) {
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
export function hasDuplicateIdOnly(importErrors) {
    return importErrors?.length === 1 && importErrors[0].type === 'duplicate_id';
}

/**
 * simple function to abstract the test for only duplicate errors
 *
 * @param   {array}  importErrors  Array< type: string, value: string } >
 *
 * @return  {boolean}              true if there are only dupe errors, no validation errors
 */
export function checkDupeOnlyErrors(importErrors) {
    return (
        importErrors?.length &&
        importErrors.find((policyErrors) => {
            const hasInvalidPolicy = policyErrors.some((err) => err.type === 'invalid_policy');
            const hasDupeErrors = policyErrors.some((err) => err.type.includes('dup'));

            return hasDupeErrors && !hasInvalidPolicy;
        })
    );
}

/**
 * function to abstract checks for whether importing is currently blocked
 *
 * @param   {object}  settings  Object{ numPolicies: number,
 *                                      messageType: string
 *                                      hasDuplicateErrors: boolean,
 *                                      duplicateResolution: Object{ resolution: string,
 *                                                           newName?: string
 *                                                         }
 *                                    }
 *
 * @return  {boolean}              true if submission blocked by current state, false otherwise
 */
export function checkForBlockedSubmit({
    numPolicies,
    messageType,
    hasDuplicateErrors,
    duplicateResolution,
}) {
    return (
        numPolicies < 1 || // at least one policy must be in selected file
        messageType === 'info' || // an info message means upload has already succeeded
        (!hasDuplicateErrors && messageType === 'error') || // error without dupes means a validation error
        (hasDuplicateErrors && !isDuplicateResolved(duplicateResolution)) // dupes, not resolved by user yet
    );
}
