import React, { Component } from 'react';
import PropTypes from 'prop-types';

import TabContent from 'Components/TabContent';

class Tabs extends Component {
    static defaultProps = {
        children: [],
        className: '',
        onTabClick: null,
        default: null,
        tabClass: 'tab mt-2',
        tabActiveClass: 'tab tab-active bg-base-100 border-t-2 mt-2',
        tabDisabledClass: 'tab disabled mt-2',
        tabContentBgColor: 'bg-base-100'
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
        className: PropTypes.string,
        onTabClick: PropTypes.func,
        default: PropTypes.shape({}),
        tabClass: PropTypes.string,
        tabActiveClass: PropTypes.string,
        tabDisabledClass: PropTypes.string,
        tabContentBgColor: PropTypes.string
    };

    constructor(props) {
        super(props);

        const index = props.headers.indexOf(props.default);

        this.state = {
            activeIndex: index === -1 ? 0 : index
        };
    }

    getHeaders() {
        const { activeIndex } = this.state;
        return this.props.headers.map((header, i) => {
            let tabClass = activeIndex === i ? this.props.tabActiveClass : this.props.tabClass;
            if (header.disabled) tabClass = this.props.tabDisabledClass;
            return (
                <button
                    className={`${tabClass} ${i === 0 ? 'ml-3' : ''}`}
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
        if (this.props.onTabClick) this.props.onTabClick(header);
        this.setState({ activeIndex: i });
    };

    renderChildren() {
        const children = React.Children.toArray(this.props.children);
        return children[this.state.activeIndex];
    }

    render() {
        return (
            <div className="w-full h-full bg-white flex flex-col">
                <div className={`flex shadow-underline font-bold ${this.props.className}`}>
                    {this.getHeaders()}
                </div>
                <div className={`overflow-hidden h-full flex-1 ${this.props.tabContentBgColor}`}>
                    {this.renderChildren()}
                </div>
            </div>
        );
    }
}

export default Tabs;
