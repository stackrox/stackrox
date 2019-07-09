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
            'tracking-wide font-700 hover:text-base-600 px-2 text-base-500 text-sm uppercase border-r border-l border-t border-base-400 rounded-t-sm',
        tabActiveClass:
            'tracking-wide bg-base-100 font-700 text-sm uppercase px-2 py-3 border-r border-l border-t border-base-400 rounded-t-sm',
        tabDisabledClass:
            'disabled tracking-wide bg-base-100 font-700 px-2 px-3 py-3 text-base-500 text-sm uppercase',
        tabContentBgColor: 'bg-base-200 border-r border-l border-b border-base-400',
        hasTabSpacing: false
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
        tabContentBgColor: PropTypes.string,
        hasTabSpacing: PropTypes.bool
    };

    // If the number of tabs reduces to being less than the active index,
    // we select the first of the remaining tabs by default.
    static getDerivedStateFromProps = (props, state) => {
        if (state.activeIndex === 0) {
            return null;
        }
        if (props.headers.length - 1 < state.activeIndex) {
            return { activeIndex: 0 };
        }
        return null;
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
        const { headers, tabActiveClass, tabClass, tabDisabledClass, hasTabSpacing } = this.props;
        return headers.map((header, i) => {
            let className = activeIndex === i ? tabActiveClass : tabClass;
            if (header.disabled) className = tabDisabledClass;
            return (
                <button
                    type="button"
                    className={`${className} ${hasTabSpacing && i !== 0 && 'ml-2'}`}
                    key={`${header.text}`}
                    onClick={this.tabClickHandler(header, i)}
                    data-test-id="tab"
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
            <div className="w-full h-full flex flex-col">
                <div className={`flex z-1 shadow-underline font-700 ${this.props.className}`}>
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
