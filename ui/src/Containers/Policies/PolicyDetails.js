import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import fieldsMap from 'Containers/Policies/policyViewDescriptors';
import policyFormFields from 'Containers/Policies/policyCreationFormDescriptor';
import { removeEmptyFields } from 'Containers/Policies/policyFormUtils';

class PolicyDetails extends Component {
    static propTypes = {
        policy: PropTypes.shape({
            fields: PropTypes.shape({}).isRequired
        }).isRequired,
        // 'notifiers' prop is being used indirectly
        // eslint-disable-next-line  react/no-unused-prop-types
        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired
            })
        ).isRequired
    };

    renderFields = () => {
        const { policy } = this.props;
        const fields = Object.keys(policy);
        if (!fields) return '';
        return (
            <div className="px-3 py-4 border-b border-base-300 bg-base-100">
                <div className="bg-white border border-base-200 shadow">
                    <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">
                        Policy Details
                    </div>
                    <div className="h-full p-3">
                        {fields.map(field => {
                            if (!fieldsMap[field]) return '';
                            const { label } = fieldsMap[field];
                            const value = fieldsMap[field].formatValue(policy[field], this.props);
                            if (!value) return '';
                            return (
                                <div className="mb-4" key={field}>
                                    <div className="py-2 text-primary-500">{label}</div>
                                    <div className="flex">{value}</div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>
        );
    };

    renderPolicyConfigurationFields = () => {
        const { policy } = this.props;
        const fields = removeEmptyFields(policy.fields);
        const fieldKeys = Object.keys(fields);
        if (!fieldKeys.length) return '';
        return (
            <div className="px-3 py-4 bg-base-100">
                <div className="bg-white border border-base-200 shadow">
                    <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">
                        {policyFormFields.policyConfiguration.header}
                    </div>
                    <div className="h-full p-3">
                        {fieldKeys.map(key => {
                            if (!fieldsMap[key]) return '';
                            const { label } = fieldsMap[key];
                            const value = fieldsMap[key].formatValue(policy.fields[key]);
                            return (
                                <div className="mb-4" key={key}>
                                    <div className="py-2 text-primary-500">{label}</div>
                                    <div className="flex">{value}</div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>
        );
    };

    render() {
        const { policy } = this.props;
        if (!policy) return null;

        return (
            <div className="flex flex-1 flex-col">
                {this.renderFields()}
                {this.renderPolicyConfigurationFields()}
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    notifiers: selectors.getNotifiers
});

export default connect(mapStateToProps)(PolicyDetails);
