import React, { ReactElement, useCallback, useEffect } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import { Link } from 'react-router-dom';
import isEqual from 'lodash/isEqual';
import {
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Spinner,
    Text,
    Title,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useTableSelection from 'hooks/useTableSelection';
import usePermissions from 'hooks/usePermissions';
import { clustersBasePath } from 'routePaths';
import { ClusterScopeObject } from 'services/RolesService';

import { ScanConfigFormValues } from './useFormikScanConfig';

export type ClusterSelectionProps = {
    clusters: ClusterScopeObject[];
    isFetchingClusters: boolean;
};

function InstallClustersButton() {
    return (
        <Link to={clustersBasePath}>
            <Button variant="primary">Install cluster</Button>
        </Link>
    );
}

function ClusterSelection({ clusters, isFetchingClusters }: ClusterSelectionProps): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');
    const { setFieldValue, values: formikValues }: FormikContextType<ScanConfigFormValues> =
        useFormikContext();

    const clusterIsPreSelected = useCallback(
        (row) => formikValues.clusters.includes(row.id),
        [formikValues.clusters]
    );

    const { allRowsSelected, selected, onSelect, onSelectAll } = useTableSelection(
        clusters,
        clusterIsPreSelected
    );

    useEffect(() => {
        const clusterIds = clusters.map((cluster) => cluster.id);
        const selectedClusterIds = clusterIds.filter((_, index) => selected[index]);
        if (!isEqual(selectedClusterIds, formikValues.clusters)) {
            setFieldValue('clusters', selectedClusterIds);
        }
    }, [selected, formikValues.clusters, setFieldValue, clusters]);

    function renderTableContent() {
        return clusters?.map(({ id, name }, rowIndex) => (
            <Tr key={id}>
                <Td
                    key={id}
                    select={{
                        rowIndex,
                        onSelect,
                        isSelected: selected[rowIndex],
                    }}
                />
                <Td>{name}</Td>
            </Tr>
        ));
    }

    function renderLoadingContent() {
        return (
            <Tr>
                <Td>
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                </Td>
            </Tr>
        );
    }

    function renderEmptyContent() {
        return (
            <Tr>
                <Td>
                    <Bullseye>
                        <EmptyStateTemplate
                            title="No clusters found"
                            headingLevel="h2"
                            icon={SearchIcon}
                        >
                            {hasWriteAccessForCluster && (
                                <Flex direction={{ default: 'column' }}>
                                    <FlexItem>
                                        <Text>Install a cluster to get started</Text>
                                    </FlexItem>
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
        if (clusters && clusters.length === 0) {
            return renderEmptyContent();
        }
        return null;
    }

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Clusters</Title>
                    </FlexItem>
                    <FlexItem>Select clusters to be included in the scan</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-u-py-lg pf-u-px-lg">
                <TableComposable variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            <Th>Name</Th>
                        </Tr>
                    </Thead>
                    <Tbody>{renderTableBodyContent()}</Tbody>
                </TableComposable>
            </Form>
        </>
    );
}

export default ClusterSelection;
