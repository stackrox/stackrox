import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTooltip from 'react-tooltip';

class Panel extends Component {
    static defaultProps = {
        header: ' ',
        buttons: [],
        width: 'w-full',
        children: []
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
        children: PropTypes.node
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
            <span key={button.text}>
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

    render() {
        return (
            <div className={`flex flex-col bg-white border border-base-300 ${this.props.width}`}>
                <div className="flex shadow-underline font-bold bg-primary-100 p-3">
                    <div className="flex flex-1 text-base-600 uppercase items-center tracking-wide">
                        {this.props.header}
                    </div>
                    <div className="flex items-center">{this.renderButtons()}</div>
                </div>
                <div className="flex flex-1 overflow-auto transition">{this.props.children}</div>
            </div>
        );
    }
}

export default Panel;
