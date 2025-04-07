import React from 'react';
import PropTypes from 'prop-types';
import Table, { defaultHeaderClassName } from 'Components/Table';
import { Tooltip } from '@patternfly/react-core';

import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';
import { sortValue } from 'sorters/sorters';

import VulnsTable from './VulnsTable';

const CVETable = (props) => {
    function getColumns() {
        const columns = [
            {
                expander: true,
                headerClassName: `w-1/8 ${defaultHeaderClassName} pointer-events-none bg-primary-200`,
                className: 'w-1/8 flex items-center justify-end',
                Expander: ({ isExpanded, ...rest }) => {
                    if (rest.original.vulns.length === 0) {
                        return '';
                    }
                    const className = 'rt-expander w-1 pt-2 pointer-events-auto';
                    return <div className={`${className} ${isExpanded ? '-open' : ''}`} />;
                },
            },
            {
                Header: 'Name',
                accessor: 'name',
                headerClassName:
                    'pl-3 font-700 text-left border-b border-base-300 border-r-0 bg-primary-200',
                Cell: ({ value }) => <div>{value}</div>,
            },
            {
                Header: 'Version',
                accessor: 'version',
                className: 'w-1/8 pr-4 flex items-center justify-end',
                headerClassName:
                    'w-1/8 font-700 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200',
            },
            {
                Header: 'Source',
                accessor: 'source',
                className: 'pr-4 flex items-center justify-end w-1/8',
                headerClassName:
                    'w-1/8 font-700 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200',
            },
            {
                Header: 'Location',
                accessor: 'location',
                className: 'flex items-center justify-start word-break-all w-1/4',
                headerClassName:
                    'w-1/4 font-700 border-b border-base-300 border-r-0 bg-primary-200',
                Cell: ({ value }) => (
                    <Tooltip content={value}>
                        <div>{value}</div>
                    </Tooltip>
                ),
            },
            {
                Header: 'CVEs',
                accessor: 'vulns.length',
                className: 'w-1/10 pr-4 flex items-center justify-end',
                headerClassName:
                    'w-1/10 font-700 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200',
            },
        ];

        if (props.containsFixableCVEs) {
            columns.push({
                Header: 'Fixable',
                className: 'w-1/10 pr-4 flex items-center justify-end',
                headerClassName:
                    'w-1/10 font-700 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200',
                Cell: ({ original }) => {
                    return original.vulns.filter((vuln) => vuln.fixedBy).length;
                },
                sortMethod: sortValue,
            });
        }
        return columns;
    }

    function renderVulnsTable({ original }) {
        const { vulns, source } = original;
        if (vulns.length === 0) {
            return null;
        }
        return (
            <VulnsTable
                vulns={vulns}
                containsFixableCVEs={props.containsFixableCVEs}
                isOSPkg={source === 'OS'}
            />
        );
    }

    const { scan, ...rest } = props;
    const columns = getColumns();
    if (!scan) {
        return <div className="p-3">No scanner setup for this registry</div>;
    }
    const { components } = scan;
    if (components.length === 0) {
        return <NoComponentVulnMessage />;
    }

    return (
        <Table
            defaultPageSize={components.length}
            className="cve-table"
            rows={components}
            columns={columns}
            SubComponent={renderVulnsTable}
            defaultSorted={[
                {
                    id: 'vulns.length',
                    desc: true,
                },
                {
                    id: 'name',
                },
            ]}
            {...rest}
        />
    );
};

CVETable.propTypes = {
    scan: PropTypes.shape({
        components: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    }).isRequired,
    containsFixableCVEs: PropTypes.bool,
};

CVETable.defaultProps = {
    containsFixableCVEs: false,
};

export default CVETable;
