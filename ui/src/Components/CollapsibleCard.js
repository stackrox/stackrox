import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';

class CollapsibleCard extends Component {
    static propTypes = {
        title: PropTypes.string.isRequired,
        children: PropTypes.node.isRequired,
        open: PropTypes.bool,
        titleClassName: PropTypes.string,
        renderWhenOpened: PropTypes.func,
        renderWhenClosed: PropTypes.func
    };

    static defaultProps = {
        open: true,
        titleClassName:
            'border-b border-base-300 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-100 hover:border-primary-300',
        renderWhenOpened: null,
        renderWhenClosed: null
    };

    renderTriggerElement = cardState => {
        const icons = {
            opened: <Icon.ChevronUp className="h-4 w-4" />,
            closed: <Icon.ChevronDown className="h-4 w-4" />
        };
        const { title, titleClassName } = this.props;
        return (
            <div className={titleClassName}>
                <h1 className="p-3 text-base-600 font-700 text-lg capitalize">{title}</h1>
                <div className="flex pr-3">{icons[cardState]}</div>
            </div>
        );
    };

    renderWhenOpened = () => this.renderTriggerElement('opened');

    renderWhenClosed = () => this.renderTriggerElement('closed');

    render() {
        const renderWhenOpened = this.props.renderWhenOpened
            ? this.props.renderWhenOpened
            : this.renderWhenOpened;
        const renderWhenClosed = this.props.renderWhenClosed
            ? this.props.renderWhenClosed
            : this.renderWhenClosed;
        return (
            <div className="bg-base-100 shadow text-base-600">
                <Collapsible
                    open={this.props.open}
                    trigger={renderWhenClosed()}
                    triggerWhenOpen={renderWhenOpened()}
                    transitionTime={100}
                >
                    {this.props.children}
                </Collapsible>
            </div>
        );
    }
}

export default CollapsibleCard;
