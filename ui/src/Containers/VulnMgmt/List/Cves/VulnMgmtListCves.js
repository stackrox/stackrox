import React from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import EntityList from 'Components/EntityList';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import TableCellLink from 'Components/TableCellLink';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import isGQLLoading from 'utils/gqlLoading';

import { CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'cve',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'cve'
        },
        {
            Header: `CVE`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'cve'
        },
        {
            Header: `Fixable`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { isFixable } = original;
                return isFixable ? <LabelChip text="Fixable" type="success" /> : 'No';
            },
            id: 'isFixable'
        },
        {
            Header: `CVSS score`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { cvss } = original;
                // TODO: add CVSS version beneath when available from API
                return <LabelChip text={(cvss && cvss.toFixed(1)) || ''} type="success" />;
            },
            id: 'cvss'
        },
        // TODO: enable this column after data is available from the API
        // {
        //     Header: `Env. Impact`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     accessor: 'envImpact'
        // },
        // TODO: enable this column after data is available from the API
        // {
        //     Header: `Impact score`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     accessor: 'impactScore'
        // },
        {
            Header: `Scanned`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { lastScanned } = original;
                return <DateTimeField date={lastScanned} />;
            },
            id: 'lastScanned'
        },
        // TODO: enable this column after data is available from the API
        // {
        //     Header: `Published`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     Cell: ({ original }) => {
        //         const { published } = original;
        //         return <DateTimeField text={published} />;
        //     },
        //     id: 'published'
        // },
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
                    .push(entityTypes.CONTROL)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${deploymentCount} ${pluralize('Deployment', deploymentCount)}`}
                    />
                );
            }
        },
        {
            Header: `Images`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { imageCount } = original;
                if (!imageCount) {
                    return <LabelChip text="No Images" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SUBJECT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${imageCount} ${pluralize('Image', imageCount)}`}
                    />
                );
            },
            id: 'imageCount'
        },
        {
            Header: `Components`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { componentCount } = original;
                if (!componentCount) {
                    return <LabelChip text="No Components" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${componentCount} ${pluralize('Component', componentCount)}`}
                    />
                );
            },
            id: 'componentCount'
        }
    ];
    return tableColumns;
};

const VulnMgmtCves = ({ match, location }) => {
    const CVES_QUERY = gql`
        query getDeployments${entityTypes.CVE} {
            vulnerabilities {
                ...cveListFields
            }
        }
        ${CVE_LIST_FRAGMENT}
    `;

    const { loading, error, data } = useQuery(CVES_QUERY);

    const tableColumns = buildTableColumns(match, location);

    if (isGQLLoading(loading, data)) return <Loader />;
    if (!data || !data.vulnerabilities || error)
        return <PageNotFound resourceType={entityTypes.CVE} />;

    return (
        <EntityList
            entityType={entityTypes.CVE}
            idAttribute="cve"
            rowData={data.vulnerabilities}
            tableColumns={tableColumns}
        />
    );
};

export default VulnMgmtCves;
