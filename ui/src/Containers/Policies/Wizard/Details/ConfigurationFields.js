import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import removeEmptyFields from 'utils/removeEmptyFields';

import fieldsMap from 'Containers/Policies/Wizard/Details/descriptors';
import policyFormFields from 'Containers/Policies/Wizard/Form/descriptors';

class ConfigurationFields extends Component {
    static propTypes = {
        clustersById: PropTypes.shape({}).isRequired,
        policy: PropTypes.shape({
            fields: PropTypes.shape({})
        }).isRequired
    };

    render() {
        if (!this.props.policy.fields) return '';

        const fields = removeEmptyFields(this.props.policy.fields);
        const fieldKeys = Object.keys(fields);
        if (!fieldKeys.length) return '';

        return (
            <div className="px-3 pt-5">
                <div className="bg-base-100 shadow">
                    <div className="p-3 border-b border-base-300 text-base-600 font-700 text-lg">
                        {policyFormFields.policyConfiguration.header}
                    </div>
                    <div className="h-full p-3 pb-0">
                        {fieldKeys.map(key => {
                            if (!fieldsMap[key]) return '';
                            const { label } = fieldsMap[key];
                            const value = fieldsMap[key].formatValue(this.props.policy.fields[key]);
                            return (
                                <div className="mb-4" key={key} data-test-id={key}>
                                    <div className="text-base-600 font-700">{label}:</div>
                                    <div className="flex pt-1 leading-normal">{value}</div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    clustersById: selectors.getClustersById
});

export default connect(mapStateToProps)(ConfigurationFields);
