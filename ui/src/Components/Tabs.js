import React, { Component } from 'react';
import PropTypes from 'prop-types';

import TabContent from 'Components/TabContent';

class Tabs extends Component {
    static defaultProps = {
        children: [],
        className: '',
        onTabClick: null,
        default: null,
        tabClass:
            'tab tracking-wide bg-base-100 font-700 hover:text-base-600 px-2 px-4 py-3 text-base-500 text-sm uppercase',
        tabActiveClass:
            'tab tab-active tracking-wide bg-base-200 text-primary-700 font-700 px-2 text-sm uppercase px-4 py-3',
        tabDisabledClass:
            'tab disabled tracking-wide bg-base-100 font-700 px-2 px-4 py-3 text-base-500 text-sm uppercase',
        tabContentBgColor: 'bg-base-200'
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
                    type="button"
                    className={`${tabClass}`}
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
            <div className="w-full h-full bg-base-100 flex flex-col">
                <div
                    className={`tab-row flex z-1 shadow-underline font-700 ${this.props.className}`}
                >
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
