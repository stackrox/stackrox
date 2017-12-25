import React, { Component } from 'react';
import PropTypes from 'prop-types';
import TabContent from 'Components/TabContent';

class Tabs extends Component {
    constructor(props) {
        super(props);

        this.state = {
            activeIndex: 0
        };
    }

    getHeaders() {
        const { activeIndex } = this.state;
        return this.props.headers.map((header, i) => {
            let tabClass = (activeIndex === i) ? 'tab tab-active mt-2' : 'tab mt-2';
            if (header.disabled) tabClass = 'tab disabled mt-2';
            return <button className={tabClass} key={`${header.text}`} onClick={this.tabClickHandler(header, i)}>{header.text}</button>;
        });
    }

    tabClickHandler = (header, i) => () => {
        if (header.disabled) return;
        this.setState({ activeIndex: i });
    }

    renderChildren() {
        const children = React.Children.toArray(this.props.children);
        return children.map((tabContentChild, i) =>
            React.cloneElement(tabContentChild, { active: this.state.activeIndex === i }));
    }

    render() {
        return (
            <div className="w-full bg-white flex flex-col">
                <div className={`flex shadow-underline font-bold mb-3 bg-primary-100 pl-3 ${this.props.className}`}>
                    {this.getHeaders()}
                </div>
                <div className="overflow-hidden h-full flex-1">{this.renderChildren()}</div>
            </div>
        );
    }
}

Tabs.defaultProps = {
    headers: [],
    children: [],
    className: ''
};

Tabs.propTypes = {
    headers: PropTypes.arrayOf(PropTypes.shape({
        text: PropTypes.string,
        disabled: PropTypes.bool
    })),
    children: (props, propName, componentName) => {
        const prop = props[propName];
        let error = null;
        React.Children.forEach(prop, (child) => {
            if (child.type !== TabContent) {
                error = new Error(`'${componentName}' children should be of type 'TabContent', but got '${child.type}'.`);
            }
        });
        return error;
    },
    className: PropTypes.string
};

export default Tabs;
