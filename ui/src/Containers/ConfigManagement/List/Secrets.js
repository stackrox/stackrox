import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { SECRETS as QUERY } from 'queries/secret';
import uniq from 'lodash/uniq';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import URLService from 'modules/URLService';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import List from './List';
import TableCellLink from './Link';

const buildTableColumns = (match, location) => {
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
            }
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
                    <span className="capitalize">
                        {uniq(files.map(file => file.type))
                            .join(', ')
                            .replace(/_/g, ' ')
                            .toLowerCase()}
                    </span>
                );
            }
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { namespace, id } = original;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.NAMESPACE)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={namespace} />;
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
                if (deployments.length === 1) return deployments[0].name;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.DEPLOYMENT)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${deployments.length} matches`} />;
            }
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.secrets;

const Secrets = ({ match, location, className, selectedRowId, onRowClick }) => {
    const tableColumns = buildTableColumns(match, location);
    return (
        <List
            className={className}
            query={QUERY}
            entityType={entityTypes.SECRET}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
        />
    );
};

Secrets.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Secrets.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Secrets;
