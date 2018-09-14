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
            'p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide cursor-pointer flex justify-between'
    };

    renderTriggerElement = cardState => {
        const icons = {
            opened: <Icon.ChevronUp className="h-4 w-4" />,
            closed: <Icon.ChevronDown className="h-4 w-4" />
        };
        return (
            <div className={this.props.titleClassName}>
                <h1 className="text-base font-600">{this.props.title}</h1>
                <div>{icons[cardState]}</div>
            </div>
        );
    };

    render() {
        return (
            <div className="bg-white shadow text-primary-600 border border-base-200">
                <Collapsible
                    open={this.props.open}
                    trigger={this.renderTriggerElement('closed')}
                    triggerWhenOpen={this.renderTriggerElement('opened')}
                    transitionTime={100}
                >
                    {this.props.children}
                </Collapsible>
            </div>
        );
    }
}

export default CollapsibleCard;
