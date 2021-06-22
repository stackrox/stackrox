/* eslint-disable react/jsx-no-bind */
import React, { CSSProperties, ReactElement, useState } from 'react';
import { Button, Switch } from '@patternfly/react-core';
import { AngleDownIcon, AngleUpIcon } from '@patternfly/react-icons';
import { TableComposable, Tbody, Td, Thead, Th, Tr, TreeRowWrapper } from '@patternfly/react-table';

import { EffectiveAccessScopeCluster, SimpleAccessScopeNamespace } from 'services/RolesService';

import EffectiveAccessScopeLabels from './EffectiveAccessScopeLabels';
import EffectiveAccessScopeStateIcon from './EffectiveAccessScopeStateIcon';

const infoName = {
    ariaLabel: 'Name of cluster or namespace (indented in light gray row)',
    tooltip: (
        <div>
            Name of <strong>cluster</strong>
            <br />
            or <strong>namespace</strong> (indented in light gray row)
        </div>
    ),
    tooltipProps: {
        isContentLeftAligned: true,
    },
};

const infoLabels = {
    ariaLabel:
        'Cluster labels specified in Platform Configuration; Namespace labels specified in Kubernetes',
    tooltip: (
        <div>
            <strong>Cluster</strong> labels specified in Platform Configuration
            <br />
            <strong>Namespace</strong> labels specified in Kubernetes
        </div>
    ),
    tooltipProps: {
        isContentLeftAligned: true,
        maxWidth: '24rem',
    },
};

// In 24px padding right of last cell in row.
const styleExpandCollapseButton = {
    position: 'absolute',
    top: '8px',
    right: '4px',
    lineHeight: '16px',
    paddingTop: 0,
    paddingRight: 0,
    paddingBottom: 0,
    paddingLeft: 0,
} as CSSProperties;

export type EffectiveAccessScopeTableProps = {
    counterComputing: number;
    clusters: EffectiveAccessScopeCluster[];
    includedClusters: string[];
    includedNamespaces: SimpleAccessScopeNamespace[];
    handleIncludedClustersChange: (clusterName: string, isChecked: boolean) => void;
    handleIncludedNamespacesChange: (
        clusterName: string,
        namespaceName: string,
        isChecked: boolean
    ) => void;
    hasAction: boolean;
};

function EffectiveAccessScopeTable({
    counterComputing,
    clusters,
    includedClusters,
    includedNamespaces,
    handleIncludedClustersChange,
    handleIncludedNamespacesChange,
    hasAction,
}: EffectiveAccessScopeTableProps): ReactElement {
    // The key is cluster id and value is boolean after click to expand/collapse namespaces.
    const [isExpandedCluster, setIsExpandedCluster] = useState<Record<string, boolean>>(
        Object.create(null)
    );

    // The key is cluster or namespace id and value is boolean after click to expand/collapse labels.
    const [isExpandedLabels, setIsExpandedLabels] = useState<Record<string, boolean>>(
        Object.create(null)
    );

    const isDisabled = !hasAction;

    function onCollapseNamespace() {} // required, but namespace has no children

    const rows: ReactElement[] = [];
    clusters.forEach((cluster, clusterIndex) => {
        const {
            id: clusterId,
            name: clusterName,
            state: clusterState,
            labels: clusterLabels,
            namespaces,
        } = cluster;

        const isExpanded = Boolean(isExpandedCluster[clusterId]);
        const clusterProps = {
            'aria-level': 1,
            'aria-posinset': clusterIndex + 1,
            'aria-setsize': namespaces.length,
            isExpanded,
        };

        rows.push(
            <TreeRowWrapper key={clusterId} row={{ props: clusterProps }}>
                <Td
                    dataLabel="Cluster name"
                    treeRow={{
                        onCollapse: () => {
                            setIsExpandedCluster({
                                ...isExpandedCluster,
                                [clusterId]: !isExpandedCluster[clusterId],
                            });
                        },
                        props: clusterProps,
                    }}
                >
                    {clusterName}
                </Td>
                <Td dataLabel="Cluster state">
                    <EffectiveAccessScopeStateIcon state={clusterState} isCluster />
                </Td>
                <Td dataLabel="Manual inclusion">
                    <Switch
                        aria-label="Included: cluster"
                        className="acs-m-manual-inclusion"
                        isChecked={includedClusters.includes(clusterName)}
                        isDisabled={isDisabled}
                        onChange={(isChecked) =>
                            handleIncludedClustersChange(clusterName, isChecked)
                        }
                    />
                </Td>
                <Td dataLabel="Cluster labels">
                    <EffectiveAccessScopeLabels
                        labels={clusterLabels}
                        isExpanded={Boolean(isExpandedLabels[clusterId])}
                    />
                    {Object.keys(clusterLabels).length > 1 && (
                        <Button
                            variant="plain"
                            aria-label="Expand or collapse cluster labels"
                            style={styleExpandCollapseButton}
                            onClick={() => {
                                setIsExpandedLabels({
                                    ...isExpandedLabels,
                                    [clusterId]: !isExpandedLabels[clusterId],
                                });
                            }}
                        >
                            {isExpandedLabels[clusterId] ? <AngleDownIcon /> : <AngleUpIcon />}
                        </Button>
                    )}
                </Td>
            </TreeRowWrapper>
        );

        namespaces.forEach((namespace, namespaceIndex) => {
            const {
                id: namespaceId,
                name: namespaceName,
                state: namespaceState,
                labels: namespaceLabels,
            } = namespace;

            const namespaceProps = {
                'aria-level': 2,
                'aria-posinset': namespaceIndex + 1,
                'aria-setsize': 0,
                isHidden: !isExpanded,
            };

            rows.push(
                <TreeRowWrapper
                    key={namespaceId}
                    row={{ props: namespaceProps }}
                    className="pf-u-background-color-200"
                >
                    <Td
                        dataLabel="Namespace name"
                        treeRow={{
                            onCollapse: onCollapseNamespace,
                            props: namespaceProps,
                        }}
                    >
                        {namespaceName}
                    </Td>
                    <Td dataLabel="Namespace state">
                        <EffectiveAccessScopeStateIcon state={namespaceState} isCluster={false} />
                    </Td>
                    <Td dataLabel="Manual inclusion">
                        <Switch
                            aria-label="Included: namespace"
                            className="acs-m-manual-inclusion"
                            isChecked={includedNamespaces.some(
                                (includedNamespace) =>
                                    includedNamespace.clusterName === clusterName &&
                                    includedNamespace.namespaceName === namespaceName
                            )}
                            isDisabled={isDisabled}
                            onChange={(isChecked) =>
                                handleIncludedNamespacesChange(
                                    clusterName,
                                    namespaceName,
                                    isChecked
                                )
                            }
                        />
                    </Td>
                    <Td dataLabel="Namespace labels">
                        <EffectiveAccessScopeLabels
                            labels={namespaceLabels}
                            isExpanded={Boolean(isExpandedLabels[namespaceId])}
                        />
                        {Object.keys(namespaceLabels).length > 1 && (
                            <Button
                                variant="plain"
                                aria-label="Expand or collapse namespace labels"
                                style={styleExpandCollapseButton}
                                onClick={() => {
                                    setIsExpandedLabels({
                                        ...isExpandedLabels,
                                        [namespaceId]: !isExpandedLabels[namespaceId],
                                    });
                                }}
                            >
                                {isExpandedLabels[namespaceId] ? (
                                    <AngleDownIcon />
                                ) : (
                                    <AngleUpIcon />
                                )}
                            </Button>
                        )}
                    </Td>
                </TreeRowWrapper>
            );
        });
    });

    return (
        <TableComposable variant="compact" isStickyHeader isTreeTable style={{ overflow: 'auto' }}>
            <Thead>
                <Tr>
                    <Th info={infoName}>Name</Th>
                    <Th
                        modifier="fitContent"
                        className={counterComputing === 0 ? '' : '--pf-global--Color--200'}
                    >
                        Status
                    </Th>
                    <Th modifier="fitContent">Manual inclusion</Th>
                    <Th info={infoLabels}>Labels</Th>
                </Tr>
            </Thead>
            <Tbody>{rows}</Tbody>
        </TableComposable>
    );
}

export default EffectiveAccessScopeTable;
