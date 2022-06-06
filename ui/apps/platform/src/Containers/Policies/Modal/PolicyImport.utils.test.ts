// system under test (SUT)
import { ImportPoliciesResponse, ImportPolicyError } from 'services/PoliciesService';
import {
    parsePolicyImportErrors,
    isDuplicateResolved,
    getResolvedPolicies,
    getErrorMessages,
    hasDuplicateIdOnly,
    checkForBlockedSubmit,
    PolicyResolutionType,
    PolicyImportErrorDuplicateName,
    PolicyImportErrorDuplicateId,
    PolicyImportErrorInvalidPolicy,
} from './PolicyImport.utils';

describe('PolicyImport.utils', () => {
    describe('parsePolicyImportErrors', () => {
        it('should return no errors when none present in given policies', () => {
            const response = getPolicy();

            const error = parsePolicyImportErrors(response.responses);

            expect(error).toEqual([]);
        });

        it('should return name error when present in given policies', () => {
            const errors = { name: true };
            const response = getPolicy(errors);

            const errorList = parsePolicyImportErrors(response.responses);
            const duplicateName =
                response.responses[0].errors[0].type === 'duplicate_name'
                    ? response.responses[0].errors[0].duplicateName
                    : null;

            expect(errorList).toEqual([
                [
                    {
                        duplicateName,
                        type: 'duplicate_name',
                        message: 'Could not add policy due to name validation',
                    },
                ],
            ]);
        });

        it('should return ID error when present in given policies', () => {
            const errors = { id: true };
            const response = getPolicy(errors);

            const errorList = parsePolicyImportErrors(response.responses);
            const duplicateName =
                response.responses[0].errors[0].type === 'duplicate_id'
                    ? response.responses[0].errors[0].duplicateName
                    : null;

            expect(errorList).toEqual([
                [
                    {
                        duplicateName,
                        incomingId: response.responses[0].policy.id,
                        incomingName: response.responses[0].policy.name,
                        type: 'duplicate_id',
                        message:
                            'Policy Fixable CVSS >= 9 (f09f8da1-6111-4ca0-8f49-294a76c65117) cannot be added because it already exists',
                    },
                ],
            ]);
        });

        it('should return both name and ID errors when present in given policies', () => {
            const errors = { name: true, id: true };
            const response = getPolicy(errors);

            const errorList = parsePolicyImportErrors(response.responses);
            const duplicateName =
                response.responses[0].errors[0].type === 'duplicate_name'
                    ? response.responses[0].errors[0].duplicateName
                    : null;

            expect(errorList).toEqual([
                [
                    {
                        duplicateName,
                        type: 'duplicate_name',
                        message: 'Could not add policy due to name validation',
                    },
                    {
                        duplicateName,
                        incomingId: response.responses[0].policy.id,
                        incomingName: response.responses[0].policy.name,
                        type: 'duplicate_id',
                        message:
                            'Policy Fixable CVSS >= 9 (f09f8da1-6111-4ca0-8f49-294a76c65117) cannot be added because it already exists',
                    },
                ],
            ]);
        });
    });

    describe('isDuplicateResolved', () => {
        it('should return false for a pristine (yet-to-be-resolved) object', () => {
            const resolutionObj = { resolution: null, newName: '' };

            const isResolved = isDuplicateResolved(resolutionObj);

            expect(isResolved).toBe(false);
        });

        it('should return false if rename is chosen, but name is empty', () => {
            const resolutionObj = { resolution: 'rename' as PolicyResolutionType, newName: '' };

            const isResolved = isDuplicateResolved(resolutionObj);

            expect(isResolved).toBe(false);
        });

        it('should return false if rename is chosen, but name is too short', () => {
            const resolutionObj = { resolution: 'rename' as PolicyResolutionType, newName: '1234' };

            const isResolved = isDuplicateResolved(resolutionObj);

            expect(isResolved).toBe(false);
        });

        it('should return true if rename is chosen, and name is minimum length', () => {
            const resolutionObj = {
                resolution: 'rename' as PolicyResolutionType,
                newName: '12345',
            };

            const isResolved = isDuplicateResolved(resolutionObj);

            expect(isResolved).toBe(true);
        });

        it('should return true if overwrite is chosen', () => {
            const resolutionObj = { resolution: 'overwrite' as PolicyResolutionType, newName: '' };

            const isResolved = isDuplicateResolved(resolutionObj);

            expect(isResolved).toBe(true);
        });
    });

    describe('getErrorMessages', () => {
        it('should return empty array for an empty error array', () => {
            const policyErrors = [];

            const errStr = getErrorMessages(policyErrors);

            expect(errStr).toEqual([]);
        });

        it('should return a message for policy name error', () => {
            const policyErrors = [
                {
                    duplicateName: 'A policy name',
                    incomingId: '1234-5678-9012-3456',
                    incomingName: 'A policy name',
                    type: 'duplicate_name',
                    message: '',
                } as PolicyImportErrorDuplicateName,
            ];

            const errStr = getErrorMessages(policyErrors);

            expect(errStr).toEqual([
                {
                    msg: 'An existing policy has the same name, “A policy name”, as the one you are trying to import.',
                    type: 'duplicate_name',
                },
            ]);
        });

        it('should return a message for policy ID error', () => {
            const policyErrors = [
                {
                    duplicateName: 'Another policy name',
                    incomingId: '1234-5678-9012-3456',
                    incomingName: 'A policy name',
                    type: 'duplicate_id',
                    message: '',
                } as PolicyImportErrorDuplicateId,
            ];

            const errStr = getErrorMessages(policyErrors);

            expect(errStr).toEqual([
                {
                    msg: 'An existing policy with the name “Another policy name” has the same ID—1234-5678-9012-3456—as the policy “A policy name” you are trying to import.',
                    type: 'duplicate_id',
                },
            ]);
        });

        it('should return two messages for policy name and policy ID errors', () => {
            const policyErrors = [
                {
                    duplicateName: 'A policy name',
                    type: 'duplicate_name',
                } as PolicyImportErrorDuplicateName,
                {
                    duplicateName: 'Another policy name',
                    incomingId: '1234-5678-9012-3456',
                    incomingName: 'A policy name',
                    type: 'duplicate_id',
                } as PolicyImportErrorDuplicateId,
            ];

            const errStr = getErrorMessages(policyErrors);

            expect(errStr).toEqual([
                {
                    msg: 'An existing policy has the same name, “A policy name”, as the one you are trying to import.',
                    type: 'duplicate_name',
                },
                {
                    msg: 'An existing policy with the name “Another policy name” has the same ID—1234-5678-9012-3456—as the policy “A policy name” you are trying to import.',
                    type: 'duplicate_id',
                },
            ]);
        });

        it('should return a message for invalid policy error', () => {
            const policyErrors = [
                {
                    type: 'invalid_policy',
                    message: 'Invalid policy',
                    validationError:
                        'policy invalid error: error validating lifecycle stage error: deploy time policy cannot contain runtime fields',
                } as PolicyImportErrorInvalidPolicy,
            ];

            const errStr = getErrorMessages(policyErrors);

            expect(errStr).toEqual([
                {
                    msg: 'Invalid policy: policy invalid error: error validating lifecycle stage error: deploy time policy cannot contain runtime fields',
                    type: 'invalid_policy',
                },
            ]);
        });
    });

    describe('getResolvedPolicies', () => {
        it('should just return the policies array as-is if there are no errors', () => {
            const response = getPolicy();
            const policies = [response.responses[0].policy];
            const errors = [];
            const duplicateResolution = { resolution: null, newName: '' };

            const [resolvedPolicies, metadata] = getResolvedPolicies(
                policies,
                errors,
                duplicateResolution
            );

            expect(resolvedPolicies).toEqual(policies);
            expect(metadata).toEqual({ overwrite: false });
        });

        it('should return metadata object with overwrite, if errors are present and overwrite is selected', () => {
            const response = getPolicy();
            const policies = [response.responses[0].policy];
            const errors = [
                {
                    duplicateName: 'A policy name',
                    type: 'duplicate_name',
                } as PolicyImportErrorDuplicateName,
            ];
            const duplicateResolution = {
                resolution: 'overwrite' as PolicyResolutionType,
                newName: '',
            };

            const [resolvedPolicies, metadata] = getResolvedPolicies(
                policies,
                errors,
                duplicateResolution
            );

            expect(resolvedPolicies).toEqual(policies);
            expect(metadata).toEqual({ overwrite: true });
        });

        it('should return policy with new name, if name error present and rename is selected', () => {
            const coolNewName = 'Bad CVEs that can be fixed';

            const response = getPolicy();
            const policies = [response.responses[0].policy];
            const errors = [
                {
                    duplicateName: 'A policy name',
                    type: 'duplicate_name',
                } as PolicyImportErrorDuplicateName,
            ];
            const duplicateResolution = {
                resolution: 'rename' as PolicyResolutionType,
                newName: coolNewName,
            };

            const [resolvedPolicies, metadata] = getResolvedPolicies(
                policies,
                errors,
                duplicateResolution
            );

            expect(resolvedPolicies[0].name).toEqual(coolNewName);
            expect(resolvedPolicies[0].id).toEqual(policies[0].id);
            expect(metadata).toEqual({ overwrite: false });
        });

        it('should return metadata object with rename, if name AND ID errors present and rename is selected', () => {
            const coolNewName = 'New equally strict policy';

            const response = getPolicy();
            const policies = [response.responses[0].policy];
            const errors = [
                {
                    duplicateName: 'A policy name',
                    type: 'duplicate_name',
                } as PolicyImportErrorDuplicateName,
                {
                    duplicateName: 'Another policy name',
                    incomingId: '1234-5678-9012-3456',
                    incomingName: 'A policy name',
                    type: 'duplicate_id',
                } as PolicyImportErrorDuplicateId,
            ];
            const duplicateResolution = {
                resolution: 'rename' as PolicyResolutionType,
                newName: coolNewName,
            };

            const [resolvedPolicies, metadata] = getResolvedPolicies(
                policies,
                errors,
                duplicateResolution
            );

            expect(resolvedPolicies[0].name).toEqual(coolNewName);
            expect(resolvedPolicies[0].id).toEqual('');
            expect(metadata).toEqual({ overwrite: false });
        });
    });

    describe('hasDuplicateIdOnly', () => {
        it('should return false if there are no errors', () => {
            const errors = [];

            const onlyDupeId = hasDuplicateIdOnly(errors);

            expect(onlyDupeId).toBe(false);
        });

        it('should return false if there is only a duplicate name error', () => {
            const errors = [
                {
                    type: 'duplicate_name',
                    message: 'Really strict policy',
                } as PolicyImportErrorDuplicateName,
            ];

            const onlyDupeId = hasDuplicateIdOnly(errors);

            expect(onlyDupeId).toBe(false);
        });

        it('should return true if there is only a duplicate ID error', () => {
            const errors = [
                {
                    duplicateName: 'Another policy name',
                    incomingId: '1234-5678-9012-3456',
                    incomingName: 'A policy name',
                    type: 'duplicate_id',
                } as PolicyImportErrorDuplicateId,
            ];

            const onlyDupeId = hasDuplicateIdOnly(errors);

            expect(onlyDupeId).toBe(true);
        });

        it('should return false if there are both duplicate ID and duplicate name errors', () => {
            const errors = [
                {
                    duplicateName: 'A policy name',
                    type: 'duplicate_name',
                } as PolicyImportErrorDuplicateName,
                {
                    duplicateName: 'Another policy name',
                    incomingId: '1234-5678-9012-3456',
                    incomingName: 'A policy name',
                    type: 'duplicate_id',
                } as PolicyImportErrorDuplicateId,
            ];

            const onlyDupeId = hasDuplicateIdOnly(errors);

            expect(onlyDupeId).toBe(false);
        });
    });

    describe('checkForBlockedSubmit', () => {
        it('should return true if no policies selected yet', () => {
            const policies = [];
            const messageObj = { message: '', type: '' };
            const duplicateErrors = null;
            const duplicateResolution = { resolution: null, newName: '' };

            const isBlocked = checkForBlockedSubmit({
                numPolicies: policies?.length || 0,
                messageType: messageObj.type,
                hasDuplicateErrors: !!duplicateErrors,
                duplicateResolution,
            });

            expect(isBlocked).toBe(true);
        });

        it('should return false if nothing blocks policy submission', () => {
            const policies = [{ name: 'Snafu' }];
            const messageObj = { message: '', type: '' };
            const duplicateErrors = null;
            const duplicateResolution = { resolution: null, newName: '' };

            const isBlocked = checkForBlockedSubmit({
                numPolicies: policies?.length || 0,
                messageType: messageObj?.type,
                hasDuplicateErrors: !!duplicateErrors,
                duplicateResolution,
            });

            expect(isBlocked).toBe(false);
        });

        it('should return true if info message means successful submission', () => {
            const policies = [{ name: 'Snafu' }];
            const messageObj = { type: 'info' };
            const duplicateErrors = null;
            const duplicateResolution = { resolution: null, newName: '' };

            const isBlocked = checkForBlockedSubmit({
                numPolicies: policies?.length || 0,
                messageType: messageObj?.type,
                hasDuplicateErrors: !!duplicateErrors,
                duplicateResolution,
            });

            expect(isBlocked).toBe(true);
        });

        it('should return true if message is error, even if no dupes', () => {
            const policies = [{ name: 'Snafu' }];
            const messageObj = { type: 'error' };
            const duplicateErrors = null;
            const duplicateResolution = { resolution: null, newName: '' };

            const isBlocked = checkForBlockedSubmit({
                numPolicies: policies?.length || 0,
                messageType: messageObj?.type,
                hasDuplicateErrors: !!duplicateErrors,
                duplicateResolution,
            });

            expect(isBlocked).toBe(true);
        });

        it('should return true if dupe error, with no resolution yet', () => {
            const policies = [{ name: 'CVE >= 7' }];
            const messageObj = { type: 'error' };
            const duplicateErrors = [{ dupeName: 'CVE >= 7' }];
            const duplicateResolution = { resolution: null, newName: '' };

            const isBlocked = checkForBlockedSubmit({
                numPolicies: policies?.length || 0,
                messageType: messageObj?.type,
                hasDuplicateErrors: !!duplicateErrors,
                duplicateResolution,
            });

            expect(isBlocked).toBe(true);
        });

        it('should return false if dupe error, but has been resolved', () => {
            const policies = [{ name: 'CVE >= 7' }];
            const messageObj = { type: 'error' };
            const duplicateErrors = [{ dupeName: 'CVE >= 7' }];
            const duplicateResolution = {
                resolution: 'overwrite' as PolicyResolutionType,
                newName: '',
            };

            const isBlocked = checkForBlockedSubmit({
                numPolicies: policies?.length || 0,
                messageType: messageObj?.type,
                hasDuplicateErrors: !!duplicateErrors,
                duplicateResolution,
            });

            expect(isBlocked).toBe(false);
        });
    });
});

function getPolicy(errors: { id?: boolean; name?: boolean } = {}): ImportPoliciesResponse {
    const errorResponse = [] as ImportPolicyError[];
    if (errors.name) {
        errorResponse.push({
            message: 'Could not add policy due to name validation',
            type: 'duplicate_name',
            duplicateName: 'Fixable CVSS >= 9',
        });
    }
    if (errors.id) {
        errorResponse.push({
            message:
                'Policy Fixable CVSS >= 9 (f09f8da1-6111-4ca0-8f49-294a76c65117) cannot be added because it already exists',
            type: 'duplicate_id',
            duplicateName: 'Fixable CVSS >= 9',
        });
    }

    return {
        responses: [
            {
                succeeded: true,
                policy: {
                    id: 'f09f8da1-6111-4ca0-8f49-294a76c65117',
                    name: 'Fixable CVSS >= 9',
                    description:
                        'Alert on deployments with fixable vulnerabilities with a CVSS of at least 9',
                    rationale:
                        'Known vulnerabilities make it easier for adversaries to exploit your application. You can fix these critical-severity vulnerabilities by updating to a newer version of the affected component(s).',
                    remediation:
                        'Use your package manager to update to a fixed version in future builds or speak with your security team to mitigate the vulnerabilities.',
                    disabled: false,
                    categories: ['Vulnerability Management'],
                    lifecycleStages: ['BUILD', 'DEPLOY'],
                    exclusions: [],
                    scope: [],
                    severity: 'HIGH_SEVERITY',
                    enforcementActions: ['FAIL_BUILD_ENFORCEMENT'],
                    notifiers: [],
                    lastUpdated: null,
                    SORTName: 'Fixable CVSS >= 9',
                    SORTLifecycleStage: 'BUILD,DEPLOY',
                    SORTEnforcement: true,
                    policyVersion: '',
                    policySections: [],
                    mitreAttackVectors: [],
                    isDefault: true,
                    criteriaLocked: false,
                    mitreVectorsLocked: false,
                    eventSource: 'NOT_APPLICABLE',
                },
                errors: errorResponse,
            },
        ],
        allSucceeded: true,
    };
}
