import { useMutation } from '@apollo/client';

import {
    VulnerabilityRequest,
    ApproveVulnerabilityRequest,
    DeleteVulnerabilityRequest,
    DenyVulnerabilityRequest,
    UndoVulnerabilityRequest,
    UpdateVulnerabilityRequest,
    APPROVE_VULNERABILITY_REQUEST,
    DENY_VULNERABILITY_REQUEST,
    DELETE_VULNERABILITY_REQUEST,
    UNDO_VULNERABILITY_REQUEST,
    UPDATE_VULNERABILITY_REQUEST,
} from './vulnerabilityRequests.graphql';
import { getExpiresOnValue, getExpiresWhenFixedValue } from './utils/vulnRequestFormUtils';

export type UseRiskAcceptance = {
    requests: VulnerabilityRequest[];
};

function useRiskAcceptance({ requests }: UseRiskAcceptance) {
    const [approveVulnerabilityRequest] = useMutation(APPROVE_VULNERABILITY_REQUEST);
    const [denyVulnerabilityRequest] = useMutation(DENY_VULNERABILITY_REQUEST);
    const [deleteVulnerabilityRequest] = useMutation(DELETE_VULNERABILITY_REQUEST);
    const [undoVulnerabilityRequest] = useMutation(UNDO_VULNERABILITY_REQUEST);
    const [updateVulnerabilityRequest] = useMutation(UPDATE_VULNERABILITY_REQUEST);

    function approveVulnRequests(values) {
        const promises = requests.map((request) => {
            const variables: ApproveVulnerabilityRequest = {
                requestID: request.id,
                comment: values.comment,
            };
            return approveVulnerabilityRequest({ variables });
        });

        return Promise.all(promises)
            .then(() => {
                return Promise.resolve({
                    message: 'Successfully approved vulnerability requests',
                    isError: false,
                });
            })
            .catch((error) => {
                return Promise.reject(new Error(error.response.data.message));
            });
    }

    function denyVulnRequests(values) {
        const promises = requests.map((request) => {
            const variables: DenyVulnerabilityRequest = {
                requestID: request.id,
                comment: values.comment,
            };
            return denyVulnerabilityRequest({ variables });
        });

        return Promise.all(promises)
            .then(() => {
                return Promise.resolve({
                    message: 'Successfully denied vulnerability requests',
                    isError: false,
                });
            })
            .catch((error) => {
                return Promise.reject(new Error(error.response.data.message));
            });
    }

    function deleteVulnRequests() {
        const promises = requests.map((request) => {
            const variables: DeleteVulnerabilityRequest = {
                requestID: request.id,
            };
            return deleteVulnerabilityRequest({ variables });
        });

        return Promise.all(promises)
            .then(() => {
                return Promise.resolve({
                    message: 'Successfully cancelled vulnerability requests',
                    isError: false,
                });
            })
            .catch((error) => {
                return Promise.reject(new Error(error.response.data.message));
            });
    }

    function updateVulnRequests(formValues) {
        const { comment } = formValues;
        let expiry = {};
        const expiresWhenFixed = getExpiresWhenFixedValue(formValues.expiresOn);
        const expiresOn = getExpiresOnValue(formValues.expiresOn);
        if (expiresWhenFixed) {
            expiry = { ...expiry, expiresWhenFixed };
        }
        if (expiresOn) {
            expiry = { ...expiry, expiresOn };
        }

        const promises = requests.map((request) => {
            const variables: UpdateVulnerabilityRequest = {
                requestID: request.id,
                comment,
                expiry,
            };
            return updateVulnerabilityRequest({ variables });
        });

        return Promise.all(promises)
            .then(() => {
                return Promise.resolve({
                    message: 'Successfully updated vulnerability requests',
                    isError: false,
                });
            })
            .catch((error) => {
                return Promise.reject(new Error(error.response.data.message));
            });
    }

    function undoVulnRequests() {
        const promises = requests.map((request) => {
            const variables: UndoVulnerabilityRequest = {
                requestID: request.id,
            };
            return undoVulnerabilityRequest({ variables });
        });

        return Promise.all(promises)
            .then(() => {
                return Promise.resolve({
                    message: 'Successfully updated vulnerability requests',
                    isError: false,
                });
            })
            .catch((error) => {
                return Promise.reject(new Error(error.response.data.message));
            });
    }

    return {
        approveVulnRequests,
        denyVulnRequests,
        deleteVulnRequests,
        updateVulnRequests,
        undoVulnRequests,
    };
}

export default useRiskAcceptance;
