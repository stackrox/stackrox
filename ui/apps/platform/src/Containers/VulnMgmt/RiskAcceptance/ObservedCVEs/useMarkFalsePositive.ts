import { useMutation } from '@apollo/client';
import { FalsePositiveFormValues } from './FalsePositiveFormModal';
import { MarkFalsePositiveRequest, MARK_FALSE_POSITIVE } from '../imageVulnerabilities.graphql';
import { getScopeValue } from '../utils/vulnRequestFormUtils';

export type UseMarkFalsePositiveProps = {
    cves: string[];
    registry: string;
    remote: string;
    tag: string;
};

function useMarkFalsePositive({ cves, registry, remote, tag }: UseMarkFalsePositiveProps) {
    const [markFalsePositive] = useMutation(MARK_FALSE_POSITIVE);

    function requestFalsePositive(formValues: FalsePositiveFormValues) {
        const { comment } = formValues;
        const scope = getScopeValue(formValues.imageAppliesTo, registry, remote, tag);

        const promises = cves.map((cve) => {
            const request: MarkFalsePositiveRequest = {
                cve,
                comment,
                scope,
            };
            const variables = { request };
            return markFalsePositive({ variables });
        });

        return Promise.all(promises)
            .then(() => {
                return Promise.resolve({
                    message: 'Successfully marked vulnerability as false positive',
                    isError: false,
                });
            })
            .catch((error) => {
                return Promise.reject(new Error(error.response.data.message));
            });
    }

    return requestFalsePositive;
}

export default useMarkFalsePositive;
