/* eslint-disable no-nested-ternary */
import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import isObject from 'lodash/isObject';
import isArray from 'lodash/isArray';
import isEmpty from 'lodash/isEmpty';

import { vulnerabilitiesPlatformPath, vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import useFeatureFlags from 'hooks/useFeatureFlags';

const isNumeric = (x) => (typeof x === 'number' || typeof x === 'string') && Number(x) >= 0;

class KeyValuePairs extends Component {
    static propTypes = {
        data: PropTypes.shape({}).isRequired,
        keyValueMap: PropTypes.shape({
            label: PropTypes.string,
            className: PropTypes.string,
        }),
        isFeatureFlagEnabled: PropTypes.func.isRequired,
    };

    static defaultProps = {
        keyValueMap: {},
        isFeatureFlagEnabled: () => false,
    };

    getKeys = () => Object.keys(this.props.data);

    getNestedValue = (data) => {
        let nestedData = data;
        let keys = nestedData;
        if (isObject(nestedData)) {
            keys = Object.keys(nestedData);
            if (keys.includes('key') && keys.includes('value') && keys.length === 2) {
                const o = { [nestedData.key]: nestedData.value };
                nestedData = o;
                keys = Object.keys(o);
            }
        }

        return keys.map((key) => (
            <div className="py-2 max-w-md" key={key} data-testid={key}>
                {!isNumeric(key) ? <span className="pr-1 font-700">{key}:</span> : ''}
                {isObject(nestedData[key]) ? (
                    this.getNestedValue(nestedData[key])
                ) : (
                    <span title={nestedData[key]}>{nestedData[key].toString()}</span>
                )}
            </div>
        ));
    };

    render() {
        const keys = this.getKeys();
        const { data, isFeatureFlagEnabled } = this.props;
        const mapping = this.props.keyValueMap;
        return keys.map((key) => {
            if (!data[key] || !mapping[key] || (isObject(data[key]) && isEmpty(data[key]))) {
                return '';
            }
            const { label } = mapping[key];
            const value = mapping[key].formatValue
                ? mapping[key].formatValue(data[key])
                : data[key];
            const { className = '' } = mapping[key];
            if (!value || (Array.isArray(value) && !value.length)) {
                return '';
            }

            const usePlatformWorkloadCvePath =
                isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') &&
                typeof data === 'object' &&
                'platformComponent' in data &&
                // eslint-disable-next-line react/prop-types
                data.platformComponent;

            const vulnMgmtBasePath = usePlatformWorkloadCvePath
                ? vulnerabilitiesPlatformPath
                : vulnerabilitiesWorkloadCvesPath;

            return (
                <div
                    className="py-3 pb-2 leading-normal border-b border-base-300 last:border-b-0"
                    key={key}
                >
                    <div className="pr-1 font-700 inline">{label}:</div>
                    <span className={`flex-1 min-w-0 ${className}`}>
                        {isObject(value) || isArray(value) ? (
                            <div className="ml-2">{this.getNestedValue(value)}</div>
                        ) : label === 'Deployment ID' && typeof value === 'string' ? (
                            <Link to={`${vulnMgmtBasePath}/deployments/${value}`}>{value}</Link>
                        ) : (
                            value.toString()
                        )}
                    </span>
                </div>
            );
        });
    }
}

function KeyValuePairsHoC(props) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    return <KeyValuePairs isFeatureFlagEnabled={isFeatureFlagEnabled} {...props} />;
}

export default KeyValuePairsHoC;
