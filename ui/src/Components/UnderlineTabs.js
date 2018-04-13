import React from 'react';
import PropTypes from 'prop-types';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';

const tabClass = 'tab flex flex-1 items-center justify-center font-600';
const tabActiveClass =
    'tab flex flex-1 items-center justify-center border-b-2 border-primary-400 font-600';
const tabDisabledClass = 'tab flex flex-1 items-center justify-center font-600 disabled';
const tabContentBgColor = 'bg-white';

const UnderlineTabs = props => (
    <Tabs
        className={props.className}
        headers={props.headers}
        onTabClick={props.onTabClick}
        default={props.default}
        tabClass={tabClass}
        tabActiveClass={tabActiveClass}
        tabDisabledClass={tabDisabledClass}
        tabContentBgColor={tabContentBgColor}
    >
        {props.children}
    </Tabs>
);

UnderlineTabs.propTypes = {
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
    default: PropTypes.shape({})
};

UnderlineTabs.defaultProps = {
    children: [],
    className: '',
    onTabClick: null,
    default: null
};

export default UnderlineTabs;
