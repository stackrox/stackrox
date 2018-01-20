import React, { Component } from 'react';
import PropTypes from 'prop-types';

class Panel extends Component {
    static defaultProps = {
        header: ' ',
        buttons: [],
        width: 'w-full',
        children: []
    };

    static propTypes = {
        header: PropTypes.string,
        buttons: PropTypes.arrayOf(PropTypes.object),
        width: PropTypes.string,
        children: PropTypes.node
    };

    renderButtons() {
        if (!this.props.buttons) return '';
        return this.props.buttons.map((button, i) => (
            <button
                key={i}
                className={button.className}
                onClick={button.onClick}
                disabled={button.disabled}
            >
                { (button.renderIcon) ? <span className="flex items-center">{button.renderIcon()}</span> : '' }
                { (button.text) ? <span className="ml-3">{button.text}</span> : '' }
            </button>
        ));
    }

    render() {
        return (
            <div className={`flex flex-col bg-white border border-base-300 ${this.props.width}`}>
                <div className="flex shadow-underline font-bold bg-primary-100 p-3">
                    <div className="flex flex-1 text-base-600 uppercase items-center tracking-wide">{this.props.header}</div>
                    <div className="flex items-center">{this.renderButtons()}</div>
                </div>
                <div className="flex flex-1 overflow-auto transition">{this.props.children}</div>
            </div>
        );
    }
}

export default Panel;
