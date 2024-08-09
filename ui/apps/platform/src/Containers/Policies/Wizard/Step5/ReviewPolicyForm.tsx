import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Flex, FlexItem, Spinner, Title, Divider, Button } from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import { DryRunAlert, checkDryRun, startDryRun } from 'services/PoliciesService';
import { ClientPolicy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getServerPolicy } from '../../policies.utils';
import PolicyDetailContent from '../../Detail/PolicyDetailContent';
import PreviewViolations from './PreviewViolations';

import './ReviewPolicyForm.css';

type ReviewPolicyFormProps = {
    isBadRequest: boolean;
    policyErrorMessage: string;
    setIsBadRequest: (isBadRequest: boolean) => void;
    setIsValidOnServer: (isValidOnServer: boolean) => void;
    setPolicyErrorMessage: (message: string) => void;
};

function ReviewPolicyForm({
    isBadRequest,
    policyErrorMessage,
    setIsBadRequest,
    setIsValidOnServer,
    setPolicyErrorMessage,
}: ReviewPolicyFormProps): ReactElement {
    const { values } = useFormikContext<ClientPolicy>();

    const [showPolicyResults, setShowPolicyResults] = useState(true);
    const [isRunningDryRun, setIsRunningDryRun] = useState(false);
    const [jobIdOfDryRun, setJobIdOfDryRun] = useState('');
    const [counterToCheckDryRun, setCounterToCheckDryRun] = useState(0);
    const [checkDryRunErrorMessage, setCheckDryRunErrorMessage] = useState('');
    const [alertsFromDryRun, setAlertsFromDryRun] = useState<DryRunAlert[]>([]);

    // Start "dry run" job for preview of violations.
    useEffect(() => {
        setIsValidOnServer(false);
        setIsRunningDryRun(true);
        setPolicyErrorMessage('');
        setIsBadRequest(false);
        setCheckDryRunErrorMessage('');
        setAlertsFromDryRun([]);

        startDryRun(getServerPolicy(values))
            .then((jobId) => {
                setIsValidOnServer(true);
                setJobIdOfDryRun(jobId);
            })
            .catch((error) => {
                setIsRunningDryRun(false);
                setPolicyErrorMessage(getAxiosErrorMessage(error));
                if (error.response?.status === 400) {
                    setIsBadRequest(true);
                }
            });
    }, [setIsBadRequest, setIsValidOnServer, setPolicyErrorMessage, values]);

    // Poll "dry run" job for preview of violations.
    useEffect(() => {
        if (jobIdOfDryRun) {
            checkDryRun(jobIdOfDryRun)
                .then(({ pending, result }) => {
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
                    setCheckDryRunErrorMessage(getAxiosErrorMessage(error));
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
        <Flex
            spaceItems={{ default: 'spaceItemsNone' }}
            fullWidth={{ default: 'fullWidth' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                fullWidth={{ default: 'fullWidth' }}
            >
                <Flex
                    flex={{ default: 'flex_1' }}
                    direction={{ default: 'column' }}
                    alignSelf={{ default: 'alignSelfStretch' }}
                    className="review-policy pf-v5-u-p-lg"
                >
                    <Flex direction={{ default: 'column', xl: 'row' }}>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <Title headingLevel="h2">Review policy</Title>
                            <div className="pf-v5-u-mt-sm">
                                Review policy settings and violations.
                            </div>
                        </FlexItem>
                        <FlexItem
                            className="pf-v5-u-pr-md"
                            alignSelf={{ default: 'alignSelfCenter' }}
                        >
                            <Button
                                variant="secondary"
                                onClick={() => setShowPolicyResults(!showPolicyResults)}
                            >
                                Preview policy violations
                            </Button>
                        </FlexItem>
                    </Flex>
                    {policyErrorMessage && (
                        <Alert
                            title={isBadRequest ? 'Policy is invalid' : 'Policy request failure'}
                            component="p"
                            variant="danger"
                            isInline
                        >
                            {policyErrorMessage}
                        </Alert>
                    )}
                </Flex>
                <Divider component="div" />
                <FlexItem className="pf-v5-u-p-lg">
                    <PolicyDetailContent policy={values} isReview />
                </FlexItem>
            </Flex>
            {showPolicyResults && (
                <>
                    <Divider component="div" orientation={{ default: 'vertical' }} />
                    <Flex
                        direction={{ default: 'column' }}
                        alignSelf={{ default: 'alignSelfStretch' }}
                        className="preview-violations pf-v5-u-p-lg pf-v5-u-w-50"
                    >
                        <Title headingLevel="h2">Preview violations</Title>
                        <div className="pf-v5-u-mb-md pf-v5-u-mt-sm">
                            The policy settings you have selected will generate violations for the
                            Build or Deploy lifecycle stages. Runtime violations are not available
                            in this preview because they are generated in response to future events.
                        </div>
                        <div className="pf-v5-u-mb-md">
                            Before you save the policy, verify that the violations seem accurate.
                        </div>
                        {values.disabled && (
                            <Alert
                                className="pf-v5-u-mb-md"
                                isInline
                                variant="info"
                                title="Policy disabled"
                                component="p"
                            >
                                <p>Violations will not be generated unless the policy is enabled</p>
                            </Alert>
                        )}
                        <Divider component="div" />
                        {isRunningDryRun ? (
                            <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                                <FlexItem>
                                    <Spinner />
                                </FlexItem>
                            </Flex>
                        ) : checkDryRunErrorMessage ? (
                            <Alert
                                title="Violations request failure"
                                component="p"
                                variant="warning"
                                isInline
                            >
                                {checkDryRunErrorMessage}
                            </Alert>
                        ) : (
                            <PreviewViolations alertsFromDryRun={alertsFromDryRun} />
                        )}
                    </Flex>
                </>
            )}
        </Flex>
    );
    /* eslint-enable no-nested-ternary */
}

export default ReviewPolicyForm;
