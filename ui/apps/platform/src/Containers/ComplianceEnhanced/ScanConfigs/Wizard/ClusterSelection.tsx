import React, { ReactElement, useCallback } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import { Link } from 'react-router-dom';
import {
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
            <Button variant="link">Go to clusters</Button>
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
            .map((cluster) => cluster.id);

        setFieldValue('clusters', newSelectedIds);
    };

    const handleSelectAll = (event: React.FormEvent<HTMLInputElement>, isSelected: boolean) => {
        onSelectAll(event, isSelected);

        const newSelectedIds = isSelected ? clusters.map((cluster) => cluster.id) : [];

        setFieldValue('clusters', newSelectedIds);
    };

    function renderTableContent() {
        return clusters?.map(({ id, name }, rowIndex) => (
            <Tr key={id}>
                <Td
                    key={id}
                    select={{
                        rowIndex,
                        onSelect: (event, isSelected) => handleSelect(event, isSelected, rowIndex),
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
                <Td colSpan={2}>
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
                <Td colSpan={2}>
                    <Bullseye>
                        <EmptyStateTemplate title="No clusters" headingLevel="h3" icon={SearchIcon}>
                            {hasWriteAccessForCluster && (
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
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Clusters</Title>
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
                                    onSelect: handleSelectAll,
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
