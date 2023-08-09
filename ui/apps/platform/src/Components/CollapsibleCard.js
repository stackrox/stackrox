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
        renderWhenClosed: PropTypes.func,
        cardClassName: PropTypes.string,
        headerComponents: PropTypes.element,
        isCollapsible: PropTypes.bool,
    };

    static defaultProps = {
        open: true,
        titleClassName:
            'border-b border-base-300 leading-normal cursor-pointer flex justify-end items-center hover:bg-primary-100 hover:border-primary-300',
        renderWhenOpened: null,
        renderWhenClosed: null,
        cardClassName: 'border border-base-400',
        headerComponents: null,
        isCollapsible: true,
    };

    renderTriggerElement = (cardState) => {
        const icons = {
            opened: <Icon.ChevronUp className="h-4 w-4" />,
            closed: <Icon.ChevronDown className="h-4 w-4" />,
        };
        const { title, titleClassName, headerComponents, isCollapsible } = this.props;
        const className = isCollapsible ? titleClassName : `${titleClassName} pointer-events-none`;
        return (
            <div className={className}>
                <h3 className="flex flex-1 p-3 pb-2 text-base-600 font-700 text-lg">{title}</h3>
                {headerComponents && <div className="pointer-events-auto">{headerComponents}</div>}
                {isCollapsible && <div className="flex px-3">{icons[cardState]}</div>}
            </div>
        );
    };

    renderWhenOpened = () => this.renderTriggerElement('opened');

    renderWhenClosed = () => this.renderTriggerElement('closed');

    render() {
        const { open, renderWhenOpened, renderWhenClosed, cardClassName, isCollapsible } =
            this.props;
        return (
            <div className={`bg-base-100 text-base-600 rounded ${cardClassName}`}>
                <Collapsible
                    open={open}
                    trigger={renderWhenClosed ? renderWhenClosed() : this.renderWhenClosed()}
                    triggerWhenOpen={
                        renderWhenOpened ? renderWhenOpened() : this.renderWhenOpened()
                    }
                    transitionTime={100}
                    lazyRender
                    triggerDisabled={!isCollapsible}
                >
                    {this.props.children}
                </Collapsible>
            </div>
        );
    }
}

export default CollapsibleCard;
