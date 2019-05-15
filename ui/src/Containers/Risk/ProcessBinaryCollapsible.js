import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';

import CollapsibleCard from 'Components/CollapsibleCard';

const titleClassName =
    'border-b border-base-300 leading-normal cursor-pointer justify-between items-center hover:border-primary-300';
class ProcessBinaryCollapsible extends Component {
    static propTypes = {
        args: PropTypes.string.isRequired,
        children: PropTypes.node.isRequired
    };

    renderHeader = (backgroundClass, icon) => {
        let { args } = this.props;
        if (args === '') args = 'No Arguments';
        return (
            <div className={`${titleClassName} ${backgroundClass}`}>
                <div className="flex items-center">
                    <div className="flex pl-2">{icon}</div>
                    <div className="p-2 text-primary-800 flex flex-1 italic">
                        <Tooltip
                            overlayClassName="w-1/4 pointer-events-none"
                            placement="top"
                            overlay={<div>{args}</div>}
                        >
                            <h1 className="text-base font-600 binary-args word-break">{args}</h1>
                        </Tooltip>
                    </div>
                </div>
            </div>
        );
    };

    renderWhenOpened = () =>
        this.renderHeader('bg-primary-100', <Icon.ChevronUp className="h-4 w-4" />);

    renderWhenClosed = () =>
        this.renderHeader('bg-base-100', <Icon.ChevronDown className="h-4 w-4" />);

    render() {
        return (
            <CollapsibleCard
                title={this.props.args}
                open
                renderWhenOpened={this.renderWhenOpened}
                renderWhenClosed={this.renderWhenClosed}
                cardClassName=""
            >
                {this.props.children}
            </CollapsibleCard>
        );
    }
}

export default ProcessBinaryCollapsible;
