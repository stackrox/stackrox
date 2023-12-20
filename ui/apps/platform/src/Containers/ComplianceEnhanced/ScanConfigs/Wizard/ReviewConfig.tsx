import React from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import {
    Alert,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Stack,
    StackItem,
    Text,
    TextList,
    TextListItem,
    TextVariants,
    Title,
} from '@patternfly/react-core';

import { ClusterScopeObject } from 'services/RolesService';
import { ComplianceProfile } from 'services/ComplianceEnhancedService';

import { ScanConfigFormValues } from './useFormikScanConfig';
import {
    convertFormikParametersToSchedule,
    formatScanSchedule,
} from '../compliance.scanConfigs.utils';

export type ProfileSelectionProps = {
    clusters: ClusterScopeObject[];
    profiles: ComplianceProfile[];
    errorMessage: string;
};

function ReviewConfig({ clusters, profiles, errorMessage }: ProfileSelectionProps) {
    const { values: formikValues }: FormikContextType<ScanConfigFormValues> = useFormikContext();

    const scanSchedule = convertFormikParametersToSchedule(formikValues.parameters);
    const formattedScanSchedule = formatScanSchedule(scanSchedule);

    function findById<T extends { id: string }>(selectedIds: string[], items: T[]): T[] {
        return selectedIds
            .map((id) => items.find((item) => item.id === id))
            .filter((item): item is T => item !== undefined);
    }

    const selectedClusters = findById<ClusterScopeObject>(formikValues.clusters, clusters);
    const selectedProfiles = findById<ComplianceProfile>(formikValues.profiles, profiles);

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Review</Title>
                    </FlexItem>
                    <FlexItem>Review and create your scan configuration</FlexItem>
                    {errorMessage && (
                        <Alert
                            title={'Scan configuration request failure'}
                            variant="danger"
                            isInline
                        >
                            {errorMessage}
                        </Alert>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Stack hasGutter className="pf-u-py-lg pf-u-px-lg">
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Name
                    </Text>
                    <Text>{formikValues.parameters.name}</Text>
                </StackItem>
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Schedule
                    </Text>
                    <Text>{formattedScanSchedule}</Text>
                </StackItem>
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Clusters
                    </Text>
                    <TextList isPlain>
                        {selectedClusters.map((cluster) => (
                            <TextListItem key={cluster.id}>{cluster.name}</TextListItem>
                        ))}
                    </TextList>
                </StackItem>
                <StackItem>
                    <Text component={TextVariants.h3} className="pf-u-font-weight-bold">
                        Profiles
                    </Text>
                    <TextList isPlain>
                        {selectedProfiles.map((profile) => (
                            <TextListItem key={profile.id}>{profile.name}</TextListItem>
                        ))}
                    </TextList>
                </StackItem>
            </Stack>
        </>
    );
}

export default ReviewConfig;
