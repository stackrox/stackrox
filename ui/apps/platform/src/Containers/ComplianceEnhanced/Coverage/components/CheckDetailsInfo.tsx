import React from 'react';
import {
    Alert,
    Bullseye,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    List,
    ListItem,
    Spinner,
} from '@patternfly/react-core';

import {
    ComplianceCheckResult,
    ComplianceClusterCheckStatus,
} from 'services/ComplianceResultsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import './CheckDetailsInfo.css';

type CheckDetailsInfoProps = {
    checkDetails?: ComplianceCheckResult | ComplianceClusterCheckStatus;
    isLoading: boolean;
    error?: Error;
};

function CheckDetailsInfo({ checkDetails, isLoading, error }: CheckDetailsInfoProps) {
    if (error) {
        return (
            <Alert title="Unable to fetch check details" component="p" isInline variant="danger">
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    if (isLoading && !checkDetails) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (checkDetails) {
        const { annotations, description, instructions, labels, rationale, warnings } =
            checkDetails;
        return (
            <DescriptionList isHorizontal>
                <DescriptionListGroup>
                    <DescriptionListTerm>Description</DescriptionListTerm>
                    <DescriptionListDescription className="formatted-text">
                        {description}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Rationale</DescriptionListTerm>
                    <DescriptionListDescription className="formatted-text">
                        {rationale}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Instructions</DescriptionListTerm>
                    <DescriptionListDescription className="formatted-text">
                        {instructions}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Labels</DescriptionListTerm>
                    <DescriptionListDescription className="formatted-text">
                        <List isPlain>
                            {Object.entries(labels).map(([key, value]) => (
                                <ListItem key={key}>
                                    {key}: {value}
                                </ListItem>
                            ))}
                        </List>
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Annotations</DescriptionListTerm>
                    <DescriptionListDescription className="formatted-text">
                        <List isPlain>
                            {Object.entries(annotations).map(([key, value]) => (
                                <ListItem key={key}>
                                    {key}: {value}
                                </ListItem>
                            ))}
                        </List>
                    </DescriptionListDescription>
                </DescriptionListGroup>
                {warnings.length > 0 && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Warning(s)</DescriptionListTerm>
                        <DescriptionListDescription>
                            <List>
                                {warnings.map((warning) => (
                                    <ListItem key={warning}>{warning}</ListItem>
                                ))}
                            </List>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                )}
            </DescriptionList>
        );
    }
}

export default CheckDetailsInfo;
