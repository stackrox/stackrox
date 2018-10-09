import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import fieldsMap from 'Containers/Policies/policyViewDescriptors';
import policyFormFields from 'Containers/Policies/policyCreationFormDescriptor';
import removeEmptyFields from 'utils/removeEmptyFields';

class PolicyDetails extends Component {
    static propTypes = {
        clustersById: PropTypes.shape({}).isRequired,
        policy: PropTypes.shape({
            fields: PropTypes.shape({}).isRequired
        }).isRequired,
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
            <div className="px-3 py-4">
                <div className="bg-base-100 shadow">
                    <div className="p-3 pb-2 border-b border-base-300 text-base-600 font-700 text-lg leading-normal">
                        Policy Details
                    </div>
                    <div className="h-full p-3 pb-0">
                        {fields.map(field => {
                            if (!fieldsMap[field]) return '';
                            const { label } = fieldsMap[field];
                            const value = fieldsMap[field].formatValue(policy[field], {
                                clustersById: this.props.clustersById,
                                notifiers: this.props.notifiers
                            });
                            if (!value) return '';
                            return (
                                <div className="mb-4" key={field}>
                                    <div className="text-base-600 font-700">{label}:</div>
                                    <div className="flex pt-1 leading-normal">{value}</div>
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
            <div className="px-3 py-4">
                <div className="bg-base-100 shadow">
                    <div className="p-3 border-b border-base-300 text-base-600 font-700 text-lg">
                        {policyFormFields.policyConfiguration.header}
                    </div>
                    <div className="h-full p-3 pb-0">
                        {fieldKeys.map(key => {
                            if (!fieldsMap[key]) return '';
                            const { label } = fieldsMap[key];
                            const value = fieldsMap[key].formatValue(policy.fields[key]);
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
    };

    render() {
        const { policy } = this.props;
        if (!policy) return null;

        return (
            <div className="flex flex-col w-full bg-base-200 overflow-auto">
                {this.renderFields()}
                {this.renderPolicyConfigurationFields()}
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    clustersById: selectors.getClustersById,
    notifiers: selectors.getNotifiers
});

export default connect(mapStateToProps)(PolicyDetails);
