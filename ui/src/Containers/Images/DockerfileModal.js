import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import Table, {
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName
} from 'Components/Table';
import Modal from 'Components/Modal';
import cloneDeep from 'lodash/cloneDeep';

import CVETable from './CVETable';

class DockerfileModal extends Component {
    static propTypes = {
        image: PropTypes.shape().isRequired,
        onClose: PropTypes.func.isRequired
    };

    renderHeader = () => (
        <header className="flex items-center w-full p-4 bg-primary-500 text-base-100 uppercase">
            <span className="flex flex-1 uppercase">Dockerfile</span>
            <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.props.onClose} />
        </header>
    );

    renderCVEsTable = row => {
        const layer = row.original;
        if (!layer.components || layer.components.length === 0) {
            return null;
        }
        return (
            <CVETable
                components={layer.components}
                containsFixableCVEs={this.props.image.fixableCves > 0}
                className="cve-table my-3 ml-4 px-2 border-0 border-l-4 border-base-300"
            />
        );
    };

    renderTable = () => {
        let extraColumns = [];

        const layers = cloneDeep(this.props.image.metadata.v1.layers);
        // If we have a scan, then we can try and assume we have layers
        if (this.props.image.scan) {
            layers.forEach((layer, i) => {
                layers[i].cvesCount = layer.components.reduce((cnt, o) => cnt + o.vulns.length, 0);
                layers[i].fixableCount = layer.components.reduce(
                    (cnt, o) => cnt + o.vulns.filter(x => x.fixedBy !== '').length,
                    0
                );
            });

            extraColumns = extraColumns.concat([
                {
                    accessor: 'components.length',
                    Header: 'Components',
                    headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
                    className: `text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`
                },
                {
                    accessor: 'cvesCount',
                    Header: 'CVEs',
                    headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
                    className: `text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`
                }
            ]);

            // Only if fixable is set, then add the column. Otherwise, it's not applicable
            if (this.props.image.fixableCves !== undefined) {
                extraColumns.push({
                    accessor: 'fixableCount',
                    Header: 'Fixable',
                    headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
                    className: `text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`
                });
            }
        }

        let columns = [
            {
                expander: true,
                headerClassName: `w-1/8 ${defaultHeaderClassName} pointer-events-none`,
                className: 'w-1/8 pointer-events-none flex items-center justify-end',
                Expander: ({ isExpanded, ...rest }) => {
                    if (rest.original.components.length === 0) return '';
                    const className = 'rt-expander w-1 pt-2 pointer-events-auto';
                    return <div className={`${className} ${isExpanded ? '-open' : ''}`} />;
                }
            },
            {
                accessor: 'instruction',
                Header: 'Instruction',
                headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
                className: `text-left pl-3 ${wrapClassName} ${defaultColumnClassName}`
            },
            {
                accessor: 'value',
                Header: 'Value',
                headerClassName: `w-3/5 text-left ${wrapClassName} ${defaultHeaderClassName}`,
                className: `w-3/5 text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`
            },
            {
                accessor: 'created',
                Header: 'Created',
                align: 'right',
                widthClassName: `text-left pr-3 ${wrapClassName} ${defaultHeaderClassName}`,
                className: `text-left pr-3 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ original }) => dateFns.format(original.created, dateTimeFormat)
            }
        ];

        if (this.props.image.scan) {
            columns = columns.concat(extraColumns);
        }

        return (
            <div className="overflow-y-scroll">
                <div className="flex flex-col w-full">
                    <Table
                        columns={columns}
                        rows={layers}
                        className="dockerfile-table"
                        defaultPageSize={layers.length}
                        SubComponent={this.renderCVEsTable}
                        showPagination={false}
                    />
                </div>
            </div>
        );
    };

    render() {
        return (
            <Modal isOpen onRequestClose={this.props.onClose} className="w-full lg:w-3/4">
                {this.renderHeader()}
                {this.renderTable()}
            </Modal>
        );
    }
}

export default DockerfileModal;
