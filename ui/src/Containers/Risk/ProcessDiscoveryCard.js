import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import CollapsibleCard from 'Components/CollapsibleCard';

const titleClassName =
    'border-b border-base-300 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-200 hover:border-primary-300';

class ProcessesDiscoveryCard extends Component {
    static propTypes = {
        name: PropTypes.string.isRequired,
        timesExecuted: PropTypes.number.isRequired,
        children: PropTypes.node.isRequired
    };

    renderHeader = (backgroundClass, icon) => {
        const { name, timesExecuted } = this.props;
        return (
            <div className={`${titleClassName} ${backgroundClass}`}>
                <div className="p-3 text-primary-800">
                    <h1 className="text-lg font-700">{name}</h1>
                    <h2 className="text-sm font-600 italic">
                        {`executed ${timesExecuted} time${timesExecuted === 1 ? '' : 's'} `}
                    </h2>
                </div>
                <div className="flex pr-3">{icon}</div>
            </div>
        );
    };

    renderWhenOpened = () =>
        this.renderHeader('bg-primary-200', <Icon.ChevronUp className="h-4 w-4" />);

    renderWhenClosed = () =>
        this.renderHeader('bg-base-100', <Icon.ChevronDown className="h-4 w-4" />);

    render() {
        return (
            <CollapsibleCard
                title={this.props.name}
                open={false}
                renderWhenOpened={this.renderWhenOpened}
                renderWhenClosed={this.renderWhenClosed}
                cardClassName="border border-base-400"
            >
                {this.props.children}
            </CollapsibleCard>
        );
    }
}

export default ProcessesDiscoveryCard;
