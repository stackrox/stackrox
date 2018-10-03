import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import CollapsibleCard from 'Components/CollapsibleCard';

class ProcessesCollapsibleCard extends Component {
    static propTypes = {
        title: PropTypes.string.isRequired,
        children: PropTypes.node.isRequired
    };

    renderCollapsibleOpened = () => {
        const titleClassName =
            'border-b border-base-300 leading-normal bg-base-200 cursor-pointer flex justify-between items-center hover:bg-primary-100 hover:border-primary-300';
        return (
            <div className={titleClassName}>
                <h1 className="p-3 text-base-600 font-700 text-lg capitalize">
                    {this.props.title}
                </h1>
                <div className="flex pr-3">
                    <Icon.ChevronUp className="h-4 w-4" />
                </div>
            </div>
        );
    };

    render() {
        return (
            <CollapsibleCard
                title={this.props.title}
                open={false}
                renderWhenOpened={this.renderCollapsibleOpened}
            >
                {this.props.children}
            </CollapsibleCard>
        );
    }
}

export default ProcessesCollapsibleCard;
