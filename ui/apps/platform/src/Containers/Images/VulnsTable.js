import React from 'react';
import PropTypes from 'prop-types';
import { Tooltip } from '@patternfly/react-core';

import Table from 'Components/Table';

const VulnsTable = ({ vulns, containsFixableCVEs, isOSPkg }) => {
    const columns = [
        {
            Header: 'CVE',
            accessor: 'cve',
            Cell: (ci) => (
                <div>
                    <a
                        href={ci.original.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary-600 font-700 pointer-events-auto"
                    >
                        {ci.value}
                    </a>
                    <div className="mt-2">{ci.original.summary}</div>
                </div>
            ),
            headerClassName: 'font-700 border-b border-base-300 flex items-end bg-primary-300',
            className: 'pointer-events-none flex items-center justify-left',
        },
        {
            Header: 'CVSS',
            accessor: 'cvss',
            width: 100,
            Cell: (ci) => {
                const cvss = ci.original && ci.original.cvss && ci.original.cvss.toFixed(1);
                if (!cvss) {
                    return (
                        <Tooltip
                            content="A CVSS value can be pending when the vulnerability has not been
                                    scored or has been disputed"
                        >
                            <div>Pending</div>
                        </Tooltip>
                    );
                }
                return `${cvss} (${ci.original.scoreVersion === 'V2' ? 'v2' : 'v3'})`;
            },
            headerClassName:
                'font-700 border-b border-base-300 flex items-end justify-end bg-primary-300',
            className: 'flex items-center justify-end',
        },
    ];
    if (containsFixableCVEs) {
        columns.push({
            Header: 'Fixed',
            accessor: 'fixedBy',
            width: 130,
            headerClassName: 'font-700 border-b border-base-300 flex items-end',
            className: 'pointer-events-none flex items-center justify-end',
            Cell: ({ value }) => (value === '' && !isOSPkg ? 'Unknown' : value),
        });
    }

    return (
        <Table
            rows={vulns}
            columns={columns}
            className="my-3 ml-4 px-2 border-0 border-l-4 border-base-300 shadow-none"
            showPagination={false}
            pageSize={vulns.length}
            defaultSorted={[
                {
                    id: 'cvss',
                    desc: true,
                },
                {
                    id: 'name',
                },
            ]}
        />
    );
};

VulnsTable.propTypes = {
    vulns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    containsFixableCVEs: PropTypes.bool,
    isOSPkg: PropTypes.bool,
};

VulnsTable.defaultProps = {
    containsFixableCVEs: false,
    isOSPkg: false,
};

export default VulnsTable;
