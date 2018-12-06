import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table, { defaultHeaderClassName } from 'Components/Table';

import VulnsTable from './VulnsTable';

class CVETable extends Component {
    static propTypes = {
        components: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        isFixable: PropTypes.bool
    };

    static defaultProps = {
        isFixable: false
    };

    getColumns = () => {
        const columns = [
            {
                expander: true,
                headerClassName: `w-1/8 ${defaultHeaderClassName} pointer-events-none bg-primary-200`,
                className: 'w-1/8 pointer-events-none flex items-center justify-end',
                Expander: ({ isExpanded, ...rest }) => {
                    if (rest.original.vulns.length === 0) return '';
                    const className = 'rt-expander w-1 pt-2 pointer-events-auto';
                    return <div className={`${className} ${isExpanded ? '-open' : ''}`} />;
                }
            },
            {
                Header: 'Name',
                accessor: 'name',
                headerClassName:
                    'pl-3 font-600 text-left border-b border-base-300 border-r-0 bg-primary-200',
                Cell: ci => <div>{ci.value}</div>
            },
            {
                Header: 'Version',
                accessor: 'version',
                className: 'pr-4 flex items-center justify-end',
                headerClassName:
                    'font-600 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200'
            },
            {
                Header: 'CVEs',
                accessor: 'vulns.length',
                className: 'w-1/8 pr-4 flex items-center justify-end',
                headerClassName:
                    'w-1/8 font-600 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200'
            }
        ];

        if (this.props.isFixable) {
            columns.push({
                Header: 'Fixable',
                accessor: 'fixableCount',
                className: 'w-1/8 pr-4 flex items-center justify-end',
                headerClassName:
                    'w-1/8 font-600 text-right border-b border-base-300 border-r-0 pr-4 bg-primary-200'
            });
        }
        return columns;
    };

    renderVulnsTable = ({ original }) => {
        const { vulns } = original;
        if (vulns.length === 0) return null;
        return <VulnsTable vulns={vulns} isFixable={this.props.isFixable} />;
    };

    render() {
        const { components, ...rest } = this.props;
        const columns = this.getColumns();
        return (
            <Table
                defaultPageSize={components.length}
                className="cve-table"
                rows={components}
                columns={columns}
                SubComponent={this.renderVulnsTable}
                defaultSorted={[
                    {
                        id: 'vulns.length',
                        desc: true
                    },
                    {
                        id: 'name'
                    }
                ]}
                {...rest}
            />
        );
    }
}

export default CVETable;
