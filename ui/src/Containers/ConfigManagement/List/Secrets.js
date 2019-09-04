import React from 'react';
import entityTypes from 'constants/entityTypes';
import { SECRETS as QUERY } from 'queries/secret';
import uniq from 'lodash/uniq';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import URLService from 'modules/URLService';
import { sortValueByLength, sortDate } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import pluralize from 'pluralize';
import List from './List';
import TableCellLink from './Link';

const secretTypeEnumMapping = {
    UNDETERMINED: 'Undetermined',
    PUBLIC_CERTIFICATE: 'Public Certificate',
    CERTIFICATE_REQUEST: 'Certificate Request',
    PRIVACY_ENHANCED_MESSAGE: 'Privacy Enhanced Message',
    OPENSSH_PRIVATE_KEY: 'OpenSSH Private Key',
    PGP_PRIVATE_KEY: 'PGP Private Key',
    EC_PRIVATE_KEY: 'EC Private Key',
    RSA_PRIVATE_KEY: 'RSA Private Key',
    DSA_PRIVATE_KEY: 'DSA Private Key',
    CERT_PRIVATE_KEY: 'Certificate Private Key',
    ENCRYPTED_PRIVATE_KEY: 'Encrypted Private Key',
    IMAGE_PULL_SECRET: 'Image Pull Secret'
};

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Secret`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Created`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { createdAt } = original;
                return format(createdAt, dateTimeFormat);
            },
            accessor: 'createdAt',
            sortMethod: sortDate
        },
        {
            Header: `File Types`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'files',
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { files } = original;
                if (!files.length) return 'No File Types';
                return (
                    <span>
                        {uniq(files.map(file => secretTypeEnumMapping[file.type])).join(', ')}
                    </span>
                );
            },
            sortMethod: sortValueByLength
        },
        entityContext && entityContext[entityTypes.CLUSTER]
            ? null
            : {
                  Header: `Cluster`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'clusterName',
                  // eslint-disable-next-line
                  Cell: ({ original, pdf }) => {
                      const { clusterName, clusterId, id } = original;
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.CLUSTER, clusterId)
                          .url();
                      return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
                  }
              },
        {
            Header: `Deployments`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'deployments',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { deployments, id } = original;
                if (!deployments.length) return 'No Deployments';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.DEPLOYMENT)
                    .url();
                const text = `${deployments.length} ${pluralize('Deployment', deployments.length)}`;
                return <TableCellLink dataTestId="deployment" pdf={pdf} url={url} text={text} />;
            },
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns.filter(col => col);
};

const createTableRows = data => {
    return data.secrets;
};

const Secrets = ({
    match,
    location,
    className,
    selectedRowId,
    onRowClick,
    query,
    data,
    entityContext
}) => {
    const autoFocusSearchInput = !selectedRowId;
    const tableColumns = buildTableColumns(match, location, entityContext);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.SECRET}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'deployments',
                    desc: true
                }
            ]}
            data={data}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
Secrets.propTypes = entityListPropTypes;
Secrets.defaultProps = entityListDefaultprops;

export default Secrets;
