import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import Widget from 'Components/Widget';
import List from 'Components/List';
import { defaultColumnClassName } from 'Components/Table';

class LinkListWidget extends Component {
    static propTypes = {
        title: PropTypes.string.isRequired,
        data: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                link: PropTypes.string.isRequired
            })
        ),
        limit: PropTypes.number,
        headerComponents: PropTypes.node,
        history: PropTypes.shape({
            push: PropTypes.func
        }).isRequired
    };

    static defaultProps = {
        data: null,
        limit: null,
        headerComponents: null
    };

    onRowSelectHandler = () => ({ link }) => {
        this.props.history.push(link);
    };

    render() {
        const { title, data, limit, headerComponents } = this.props;
        const columns = [
            {
                id: 'name',
                accessor: 'name',
                className: `${defaultColumnClassName} underline`,
                Cell: ({ value }) => <div className="truncate pr-4">{value}</div>
            }
        ];
        let truncatedData = data;
        if (limit) truncatedData = data.slice(0, limit);
        return (
            <Widget
                header={title}
                headerComponents={headerComponents}
                bodyClassName="bg-base-100 flex-col"
            >
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

export default withRouter(LinkListWidget);
