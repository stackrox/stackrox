import { connect } from 'react-redux';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import Widget from 'Components/Widget';
import List from 'Components/List';
import { defaultColumnClassName, rtTrActionsClassName } from 'Components/Table';

class LinkListWidget extends Component {
    static propTypes = {
        title: PropTypes.string.isRequired,
        data: PropTypes.arrayOf(PropTypes.shape({})),
        length: PropTypes.number
    };

    static defaultProps = {
        data: null,
        length: 5
    };

    onRowSelectHandler = () => () =>
        // TODO: connect search functionality to row clicks
        null;

    renderRowActionButtons = () => (
        <div className="text-base-600 pr-1">
            <Icon.ExternalLink className="mt-1 h-4 w-4" />
        </div>
    );

    render() {
        const { title, data, length } = this.props;
        const columns = [
            {
                id: 'name',
                accessor: 'name',
                className: `${defaultColumnClassName} underline`,
                Cell: ({ value }) => <div className="truncate pr-4">{value}</div>
            },
            {
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: () => this.renderRowActionButtons()
            }
        ];
        const truncatedData = data.slice(0, length);
        return (
            <Widget header={title} bodyClassName="bg-base-100 flex-col">
                <List
                    columns={columns}
                    rows={truncatedData}
                    selectRow={this.onRowSelectHandler()}
                    selectedIdAttribute="name"
                />
            </Widget>
        );
    }
}

export default connect()(LinkListWidget);
