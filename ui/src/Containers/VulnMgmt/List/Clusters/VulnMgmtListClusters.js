import React from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import LabelChip from 'Components/LabelChip';
import EntityList from 'Components/EntityList';
import TableCellLink from 'Components/TableCellLink';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import isGQLLoading from 'utils/gqlLoading';

import { CLUSTER_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        // {
        // TODO: enable this column after data is available from the API
        //     Header: `CVEs`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     accessor: 'cves'
        // },
        {
            Header: `K8S version`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'status.orchestratorMetadata.version'
        },
        // TODO: enable this column after data is available from the API
        // {
        //     Header: `Created`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     accessor: 'createdAt'
        // },
        {
            Header: `Namespaces`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { namespaceCount } = original;
                if (!namespaceCount) {
                    return <LabelChip text="No Namespaces" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.CONTROL)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${namespaceCount} ${pluralize('Namespace', namespaceCount)}`}
                    />
                );
            }
        },
        {
            Header: `Deployments`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { deploymentCount } = original;
                if (!deploymentCount) {
                    return <LabelChip text="No Deployments" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SUBJECT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${deploymentCount} ${pluralize('Deployment', deploymentCount)}`}
                    />
                );
            },
            id: 'deploymentCount'
        },
        {
            Header: `Policies`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { policyCount } = original;
                if (!policyCount) {
                    return <LabelChip text="No Policies" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${policyCount} ${pluralize('Policy', policyCount)}`}
                    />
                );
            },
            id: 'policyCount'
        },
        {
            Header: `Policy status`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                return policyStatus.status === 'pass' ? (
                    <LabelChip text="Pass" type="success" />
                ) : (
                    <LabelChip text="Fail" type="alert" />
                );
            },
            id: 'policyStatus'
        } // ,
        // {
        //     Header: `Latest violation`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     accessor: 'latestViolation'
        // },
        // {
        //     Header: `Risk`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     accessor: 'risk'
        // }
    ];
    return tableColumns;
};

const VulnMgmtClusters = ({ match, location }) => {
    const CLUSTERS_QUERY = gql`
        query getClusters${entityTypes.CLUSTER} {
            clusters {
                ...clusterListFields
            }
        }
        ${CLUSTER_LIST_FRAGMENT}
    `;

    const { loading, error, data } = useQuery(CLUSTERS_QUERY);

    const tableColumns = buildTableColumns(match, location);

    if (isGQLLoading(loading, data)) return <Loader />;
    if (!data || !data.clusters || error)
        return <PageNotFound resourceType={entityTypes.CLUSTER} />;

    return (
        <EntityList
            entityType={entityTypes.CLUSTER}
            idAttribute="id"
            rowData={data.clusters}
            tableColumns={tableColumns}
        />
    );
};

export default VulnMgmtClusters;
