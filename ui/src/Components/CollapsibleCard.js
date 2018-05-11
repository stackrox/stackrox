import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';

class CollapsibleCard extends Component {
    static propTypes = {
        title: PropTypes.string.isRequired,
        open: PropTypes.bool,
        children: PropTypes.node.isRequired
    };

    static defaultProps = {
        open: true
    };

    renderTriggerElement = cardState => {
        const icons = {
            opened: <Icon.ChevronUp className="h-4 w-4" />,
            closed: <Icon.ChevronDown className="h-4 w-4" />
        };

        return (
            <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide cursor-pointer flex justify-between">
                <div>{this.props.title}</div>
                <div>{icons[cardState]}</div>
            </div>
        );
    };

    render() {
        return (
            <Collapsible
                open={this.props.open}
                trigger={this.renderTriggerElement('closed')}
                triggerWhenOpen={this.renderTriggerElement('opened')}
                transitionTime={100}
            >
                {this.props.children}
            </Collapsible>
        );
    }
}

export default CollapsibleCard;
