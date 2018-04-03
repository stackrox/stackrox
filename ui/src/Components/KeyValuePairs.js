import React, { Component } from 'react';
import PropTypes from 'prop-types';

import isObject from 'lodash/isObject';
import isArray from 'lodash/isArray';
import isEmpty from 'lodash/isEmpty';

class KeyValuePairs extends Component {
    static propTypes = {
        data: PropTypes.shape({}).isRequired,
        keyValueMap: PropTypes.shape({})
    };

    static defaultProps = {
        keyValueMap: {}
    };

    getKeys = () => Object.keys(this.props.data);

    getNestedValue = data => {
        let keys = data;
        if (isObject(data)) {
            keys = Object.keys(data);
        }
        return keys.map(key => (
            <div className="flex py-2" key={key}>
                <div className="pr-1">{key}:</div>
                <div className="italic text-accent-400">
                    {isObject(data[key]) ? (
                        <div>
                            <br />
                            {this.getNestedValue(data[key])}
                        </div>
                    ) : (
                        data[key]
                    )}
                </div>
            </div>
        ));
    };

    render() {
        const keys = this.getKeys();
        const { data } = this.props;
        const mapping = this.props.keyValueMap;
        return keys.map(key => {
            if (!data[key] || !mapping[key] || (isObject(data[key]) && isEmpty(data[key])))
                return '';
            const { label } = mapping[key];
            const value = mapping[key].formatValue
                ? mapping[key].formatValue(data[key])
                : data[key];
            if (!value || (Array.isArray(value) && !value.length)) return '';
            return (
                <div className="flex py-3" key={key}>
                    <div className="pr-1">{label}:</div>
                    <div className={`font-500 ${(isObject(value) || isArray(value)) && '-ml-8'}`}>
                        {isObject(value) || isArray(value) ? (
                            <div>
                                <br />
                                {this.getNestedValue(value)}
                            </div>
                        ) : (
                            value
                        )}
                    </div>
                </div>
            );
        });
    }
}

export default KeyValuePairs;
