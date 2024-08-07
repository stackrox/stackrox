import React, { ReactElement, RefObject, useCallback } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import { Link } from 'react-router-dom';
import {
    Alert,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import useTableSelection from 'hooks/useTableSelection';
import { clustersBasePath } from 'routePaths';
import { ComplianceIntegration } from 'services/ComplianceIntegrationService';

import { ScanConfigFormValues } from '../compliance.scanConfigs.utils';
import ComplianceClusterStatus from '../components/ComplianceClusterStatus';

export type ClusterSelectionProps = {
    alertRef: RefObject<HTMLDivElement>;
    clusters: ComplianceIntegration[];
    isFetchingClusters: boolean;
};

function InstallClustersButton() {
    return (
        <Link to={clustersBasePath}>
            <Button variant="link">Go to clusters</Button>
        </Link>
    );
}

function ClusterSelection({
    alertRef,
    clusters,
    isFetchingClusters,
}: ClusterSelectionProps): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForClusters = isRouteEnabled('clusters');
    const {
        setFieldValue,
        setTouched,
        values: formikValues,
        touched: formikTouched,
    }: FormikContextType<ScanConfigFormValues> = useFormikContext();

    const clusterIsPreSelected = useCallback(
        (row) => formikValues.clusters.includes(row.clusterId),
        [formikValues.clusters]
    );

    const { allRowsSelected, selected, onSelect, onSelectAll } = useTableSelection(
        clusters,
        clusterIsPreSelected
    );

    const handleSelect = (
        event: React.FormEvent<HTMLInputElement>,
        isSelected: boolean,
        rowId: number
    ) => {
        onSelect(event, isSelected, rowId);

        const newSelectedIds = clusters
            .filter((_, index) => {
                return index === rowId ? isSelected : selected[index];
            })
            .map((cluster) => cluster.clusterId);

        setTouched({ ...formikTouched, clusters: true });
        setFieldValue('clusters', newSelectedIds);
    };

    const handleSelectAll = (event: React.FormEvent<HTMLInputElement>, isSelected: boolean) => {
        onSelectAll(event, isSelected);

        const newSelectedIds = isSelected ? clusters.map((cluster) => cluster.clusterId) : [];

        setTouched({ ...formikTouched, clusters: true });
        setFieldValue('clusters', newSelectedIds);
    };

    function renderTableContent() {
        return clusters?.map(({ clusterId, clusterName, statusErrors }, rowIndex) => (
            <Tr key={clusterId}>
                <Td
                    key={clusterId}
                    select={{
                        rowIndex,
                        onSelect: (event, isSelected) => handleSelect(event, isSelected, rowIndex),
                        isSelected: selected[rowIndex],
                    }}
                />
                <Td dataLabel="Name">{clusterName}</Td>
                <Td dataLabel="Operator status">
                    <ComplianceClusterStatus errors={statusErrors} />
                </Td>
            </Tr>
        ));
    }

    function renderLoadingContent() {
        return (
            <Tr>
                <Td colSpan={2}>
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                </Td>
            </Tr>
        );
    }

    function renderEmptyContent() {
        return (
            <Tr>
                <Td colSpan={2}>
                    <Bullseye>
                        <EmptyStateTemplate title="No clusters" headingLevel="h3" icon={SearchIcon}>
                            {isRouteEnabledForClusters && (
                                <Flex direction={{ default: 'column' }}>
                                    <FlexItem>
                                        <InstallClustersButton />
                                    </FlexItem>
                                </Flex>
                            )}
                        </EmptyStateTemplate>
                    </Bullseye>
                </Td>
            </Tr>
        );
    }

    function renderTableBodyContent() {
        if (isFetchingClusters) {
            return renderLoadingContent();
        }
        if (clusters && clusters.length > 0) {
            return renderTableContent();
        }
        return renderEmptyContent();
    }

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Clusters</Title>
                    </FlexItem>
                    <FlexItem>Select clusters to be included in the scan</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-v5-u-py-lg pf-v5-u-px-lg" ref={alertRef}>
                {formikTouched.clusters && formikValues.clusters.length === 0 && (
                    <Alert
                        title="At least one cluster is required to proceed"
                        component="p"
                        variant="danger"
                        isInline
                    />
                )}
                <Table>
                    <Thead noWrap>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: handleSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            <Th>Name</Th>
                            <Th>Operator status</Th>
                        </Tr>
                    </Thead>
                    <Tbody>{renderTableBodyContent()}</Tbody>
                </Table>
            </Form>
        </>
    );
}

export default ClusterSelection;
