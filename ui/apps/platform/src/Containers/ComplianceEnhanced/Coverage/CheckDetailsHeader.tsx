import React from 'react';
import { Flex, FlexItem, Title, Skeleton, Alert, Label, LabelGroup } from '@patternfly/react-core';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ComplianceCheckResultStatusCount } from 'services/ComplianceCommon';

import { getClusterResultsStatusObject, sortCheckStats } from './compliance.coverage.utils';
import ControlLabels from './components/ControlLabels';

interface CheckDetailsHeaderProps {
    checkName: string;
    checkStatsResponse: ComplianceCheckResultStatusCount | undefined;
    isLoading: boolean;
    error: Error | undefined;
}

function CheckDetailsHeader({
    checkName,
    checkStatsResponse,
    isLoading,
    error,
}: CheckDetailsHeaderProps) {
    function renderDynamicContent() {
        if (isLoading) {
            return <Skeleton screenreaderText="Loading check stats" height="100px" />;
        }

        if (error) {
            return (
                <Alert variant="danger" title="Unable to fetch check stats" component="p" isInline>
                    {getAxiosErrorMessage(error)}
                </Alert>
            );
        }

        if (checkStatsResponse) {
            const { checkStats, controls, rationale } = checkStatsResponse;
            const sortedCheckStats = sortCheckStats(checkStats);
            return (
                <>
                    <FlexItem>
                        <LabelGroup numLabels={Infinity}>
                            {sortedCheckStats.map((checkStat) => {
                                if (checkStat.count > 0) {
                                    const statusObject = getClusterResultsStatusObject(
                                        checkStat.status
                                    );
                                    return (
                                        <Label
                                            variant="filled"
                                            icon={statusObject.icon}
                                            color={statusObject.color}
                                            key={checkStat.status}
                                        >
                                            {`${statusObject.statusText}: ${checkStat.count}`}
                                        </Label>
                                    );
                                }
                                return null;
                            })}
                        </LabelGroup>
                    </FlexItem>
                    <FlexItem>{rationale}</FlexItem>
                    <FlexItem>
                        <ControlLabels controls={controls} numLabels={6} />
                    </FlexItem>
                </>
            );
        }

        return null;
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
            <FlexItem>
                <Title headingLevel="h1" className="pf-v5-u-w-100">
                    {checkName}
                </Title>
            </FlexItem>
            {renderDynamicContent()}
        </Flex>
    );
}

export default CheckDetailsHeader;
