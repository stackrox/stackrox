import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';

class VulnsTable extends Component {
    static propTypes = {
        vulns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        containsFixableCVEs: PropTypes.bool
    };

    static defaultProps = {
        containsFixableCVEs: false
    };

    getColumns = () => {
        const columns = [
            {
                Header: 'CVE',
                accessor: 'cve',
                Cell: ci => (
                    <div className="truncate">
                        <a
                            href={ci.original.link}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-primary-600 font-600 pointer-events-auto"
                        >
                            {ci.value}
                        </a>
                        - {ci.original.summary}
                    </div>
                ),
                headerClassName: 'font-600 border-b border-base-300 flex items-end bg-primary-300',
                className: 'pointer-events-none flex items-center justify-left italic'
            },
            {
                Header: 'CVSS',
                accessor: 'cvss',
                width: 50,
                headerClassName:
                    'font-600 border-b border-base-300 flex items-end justify-end bg-primary-300',
                className: 'pointer-events-none flex items-center justify-end italic'
            }
        ];
        if (this.props.containsFixableCVEs) {
            columns.push({
                Header: 'Fixed',
                accessor: 'fixedBy',
                width: 130,
                headerClassName: 'font-600 border-b border-base-300 flex items-end',
                className: 'pointer-events-none flex items-center justify-end italic'
            });
        }
        return columns;
    };

    render() {
        const { vulns } = this.props;
        return (
            <Table
                rows={vulns}
                columns={this.getColumns()}
                className="my-3 ml-4 px-2 border-0 border-l-4 border-base-300 shadow-none"
                showPagination={false}
                pageSize={vulns.length}
                defaultSorted={[
                    {
                        id: 'cvss',
                        desc: true
                    },
                    {
                        id: 'name'
                    }
                ]}
            />
        );
    }
}

export default VulnsTable;
