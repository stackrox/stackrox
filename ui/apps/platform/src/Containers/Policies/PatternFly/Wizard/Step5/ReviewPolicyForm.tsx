import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Flex, FlexItem, Spinner, Title } from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import { DryRunAlert, checkDryRun, startDryRun } from 'services/PoliciesService';
import { Policy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import PolicyOverview from '../../Detail/PolicyOverview';
import PreviewViolations from './PreviewViolations';

import './ReviewPolicyForm.css';

function ReviewPolicyForm(): ReactElement {
    const { values } = useFormikContext<Policy>();

    const [isRunningDryRun, setIsRunningDryRun] = useState(false);
    const [jobIdOfDryRun, setJobIdOfDryRun] = useState('');
    const [errorMessageFromDryRun, setErrorMessageFromDryRun] = useState('');
    const [counterToCheckDryRun, setCounterToCheckDryRun] = useState(0);
    const [alertsFromDryRun, setAlertsFromDryRun] = useState<DryRunAlert[]>([]);

    // Start "dry run" job for preview of violations.
    useEffect(() => {
        setIsRunningDryRun(true);
        setErrorMessageFromDryRun('');
        setAlertsFromDryRun([]);

        startDryRun(values)
            .then(({ data: { jobId } }) => {
                /*
                 * TODO after policiesSagas.js has been deleted:
                 * Replace ({ data: { jobId } }) with (jobId) above.
                 */
                setJobIdOfDryRun(jobId);
            })
            .catch((error) => {
                setIsRunningDryRun(false);
                setErrorMessageFromDryRun(getAxiosErrorMessage(error));
            });
    }, [values]);

    // Poll "dry run" job for preview of violations.
    useEffect(() => {
        if (jobIdOfDryRun) {
            checkDryRun(jobIdOfDryRun)
                .then(({ data: { pending, result } }) => {
                    /*
                     * TODO after policiesSagas.js has been deleted:
                     * Replace ({ data: { pending, result } }) with ({ pending, result }) above.
                     */
                    if (pending) {
                        // To make another request, increment counterToCheckDryRun which is in useEffect dependencies.
                        setCounterToCheckDryRun((counter) => counter + 1);
                    } else {
                        setIsRunningDryRun(false);
                        setJobIdOfDryRun('');
                        setCounterToCheckDryRun(0);
                        setAlertsFromDryRun(result.alerts);
                    }
                })
                .catch((error) => {
                    setIsRunningDryRun(false);
                    setErrorMessageFromDryRun(getAxiosErrorMessage(error));
                    setJobIdOfDryRun('');
                    setCounterToCheckDryRun(0);
                });
        }
    }, [jobIdOfDryRun, counterToCheckDryRun]);

    /*
     * flex_1 so columns have equal width.
     * alignSelfStretch so columns have equal height for border.
     */

    /* eslint-disable no-nested-ternary */
    return (
        <Flex direction={{ default: 'row' }}>
            <Flex
                flex={{ default: 'flex_1' }}
                direction={{ default: 'column' }}
                alignSelf={{ default: 'alignSelfStretch' }}
                className="review-policy"
            >
                <Title headingLevel="h2">Review policy</Title>
                <div className="pf-u-mb-md pf-u-mt-sm">Review policy settings and violations.</div>
                <PolicyOverview clusters={[]} notifiers={[]} policy={values} />
            </Flex>
            <Flex
                flex={{ default: 'flex_1' }}
                direction={{ default: 'column' }}
                alignSelf={{ default: 'alignSelfStretch' }}
                className="preview-violations"
            >
                <Title headingLevel="h2">Preview violations</Title>
                <div className="pf-u-mb-md pf-u-mt-sm">
                    The policy settings you have selected will generate violations for the following
                    deployments. Before you save the policy, verify that the violations seem
                    accurate.
                </div>
                {isRunningDryRun ? (
                    <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                        <FlexItem>
                            <Spinner isSVG />
                        </FlexItem>
                    </Flex>
                ) : errorMessageFromDryRun ? (
                    <Alert title="Request failure for violations" variant="danger" isInline>
                        {errorMessageFromDryRun}
                    </Alert>
                ) : (
                    <PreviewViolations alertsFromDryRun={alertsFromDryRun} />
                )}
            </Flex>
        </Flex>
    );
    /* eslint-enable no-nested-ternary */
}

export default ReviewPolicyForm;
