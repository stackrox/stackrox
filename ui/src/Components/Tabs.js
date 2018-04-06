import React, { Component } from 'react';
import PropTypes from 'prop-types';

import TabContent from 'Components/TabContent';

class Tabs extends Component {
    static defaultProps = {
        children: [],
        className: ''
    };

    static propTypes = {
        headers: PropTypes.arrayOf(
            PropTypes.shape({
                text: PropTypes.string,
                disabled: PropTypes.bool
            })
        ).isRequired,
        children: (props, propName, componentName) => {
            const prop = props[propName];
            let error = null;
            React.Children.forEach(prop, child => {
                if (child.type !== TabContent) {
                    error = new Error(
                        `'${componentName}' children should be of type 'TabContent', but got '${
                            child.type
                        }'.`
                    );
                }
            });
            return error;
        },
        className: PropTypes.string
    };

    constructor(props) {
        super(props);

        this.state = {
            activeIndex: 0
        };
    }

    getHeaders() {
        const { activeIndex } = this.state;
        return this.props.headers.map((header, i) => {
            let tabClass =
                activeIndex === i ? 'tab tab-active bg-base-100 border-t-2 mt-2' : 'tab mt-2';
            if (header.disabled) tabClass = 'tab disabled mt-2';
            return (
                <button
                    className={tabClass}
                    key={`${header.text}`}
                    onClick={this.tabClickHandler(header, i)}
                >
                    {header.text}
                </button>
            );
        });
    }

    tabClickHandler = (header, i) => () => {
        if (header.disabled) return;
        this.setState({ activeIndex: i });
    };

    renderChildren() {
        const children = React.Children.toArray(this.props.children);
        return children[this.state.activeIndex];
    }

    render() {
        return (
            <div className="w-full bg-white flex flex-col">
                <div className={`flex shadow-underline font-bold pl-3 ${this.props.className}`}>
                    {this.getHeaders()}
                </div>
                <div className="overflow-hidden pt-3 h-full flex-1 bg-base-100">
                    {this.renderChildren()}
                </div>
            </div>
        );
    }
}

export default Tabs;
