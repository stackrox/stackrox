import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTooltip from 'react-tooltip';
import * as Icon from 'react-feather';

class Panel extends Component {
    static defaultProps = {
        header: ' ',
        buttons: [],
        width: 'w-full',
        children: [],
        onClose: null
    };

    static propTypes = {
        header: PropTypes.string,
        buttons: PropTypes.arrayOf(
            PropTypes.shape({
                className: PropTypes.string,
                onClick: PropTypes.func,
                disabled: PropTypes.bool,
                text: PropTypes.string,
                renderIcon: PropTypes.func,
                tooltip: PropTypes.string
            })
        ),
        width: PropTypes.string,
        children: PropTypes.node,
        onClose: PropTypes.func
    };

    renderToolTip = button => {
        if (!button.tooltip) return '';
        return (
            <ReactTooltip id={`button-${button.text}`} type="dark" effect="solid">
                {button.tooltip}
            </ReactTooltip>
        );
    };

    renderButtons() {
        if (!this.props.buttons) return '';
        return this.props.buttons.map(button => (
            <span key={`button_${button.text}`}>
                <button
                    className={button.className}
                    onClick={button.onClick}
                    disabled={button.disabled}
                    data-tip
                    data-for={`button-${button.text}`}
                >
                    {button.renderIcon && (
                        <span className="flex items-center">{button.renderIcon()}</span>
                    )}
                    {button.text && <span className="ml-3">{button.text}</span>}
                </button>
                {this.renderToolTip(button)}
            </span>
        ));
    }

    renderCancelButton() {
        if (!this.props.onClose) return '';
        return (
            <div className="flex items-end border-base-300 items-center hover:bg-primary-300 ml-2 border-l">
                <span>
                    <button
                        className="cancel flex text-primary-600 p-4 text-center text-sm items-center hover:text-white"
                        onClick={this.props.onClose}
                        data-tip
                        data-for="button-cancel"
                    >
                        <Icon.X className="h-4 w-4" />
                        {this.renderToolTip('Cancel')}
                    </button>
                </span>
            </div>
        );
    }

    render() {
        return (
            <div
                className={`side-panel flex flex-col bg-white border h-full border-t-0 border-base-300 ${
                    this.props.width
                }`}
            >
                <div className="shadow-underline font-bold bg-primary-100">
                    <div className="flex flex-row w-full">
                        <div className="flex flex-1 text-base-600 uppercase items-center tracking-wide py-2 px-4">
                            {this.props.header}
                        </div>
                        <div className="flex items-center py-2 px-4">{this.renderButtons()}</div>
                        {this.renderCancelButton()}
                    </div>
                </div>
                <div className="flex flex-1 overflow-auto transition">{this.props.children}</div>
            </div>
        );
    }
}

export default Panel;
