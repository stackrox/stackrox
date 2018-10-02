import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';

class CollapsibleCard extends Component {
    static propTypes = {
        title: PropTypes.string.isRequired,
        children: PropTypes.node.isRequired,
        open: PropTypes.bool,
        titleClassName: PropTypes.string
    };

    static defaultProps = {
        open: true,
        titleClassName:
            'border-b border-base-300 tracking-wide cursor-pointer flex justify-between items-center hover:bg-primary-100 hover:border-primary-300'
    };

    renderTriggerElement = cardState => {
        const icons = {
            opened: <Icon.ChevronUp className="h-4 w-4" />,
            closed: <Icon.ChevronDown className="h-4 w-4" />
        };
        return (
            <div className={this.props.titleClassName}>
                <h1 className="p-3 text-base-600 font-700 text-lg capitalize">
                    {this.props.title}
                </h1>
                <div className="flex pr-3">{icons[cardState]}</div>
            </div>
        );
    };

    render() {
        return (
            <div className="bg-base-100 shadow text-base-600">
                <Collapsible
                    open={this.props.open}
                    trigger={this.renderTriggerElement('closed')}
                    triggerWhenOpen={this.renderTriggerElement('opened')}
                    transitionTime={1}
                >
                    {this.props.children}
                </Collapsible>
            </div>
        );
    }
}

export default CollapsibleCard;
