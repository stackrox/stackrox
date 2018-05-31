import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import flatten from 'flat';
import omitBy from 'lodash/omitBy';
import difference from 'lodash/difference';
import pick from 'lodash/pick';
import fieldsMap from 'Containers/Policies/policyViewDescriptors';

const categoryGroupsMap = {
    imagePolicy: 'Image Assurance',
    configurationPolicy: 'Container Configuration',
    privilegePolicy: 'Privileges and Capabilities'
};

const categories = Object.keys(categoryGroupsMap);

class PolicyDetails extends Component {
    static propTypes = {
        policy: PropTypes.shape({}),
        // 'notifiers' prop is being used indirectly
        // eslint-disable-next-line  react/no-unused-prop-types
        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired
            })
        ).isRequired,
        history: ReactRouterPropTypes.history.isRequired
    };

    static defaultProps = {
        policy: null
    };

    removeEmptyFields = obj => {
        const flattenedObj = flatten(obj);
        const omittedObj = omitBy(
            flattenedObj,
            value => value === null || value === undefined || value === '' || value === []
        );
        const newObj = flatten.unflatten(omittedObj);
        return newObj;
    };

    updateSelectedPolicy = policy => {
        const urlSuffix = policy && policy.id ? `/${policy.id}` : '';
        this.props.history.push({
            pathname: `/main/policies${urlSuffix}`
        });
    };

    renderFields = () => {
        const policy = this.removeEmptyFields(this.props.policy);
        const fields = Object.keys(policy);
        const policyDetails = difference(fields, categories);
        if (!policyDetails) return '';
        return (
            <div className="px-3 py-4">
                <div className="bg-white border border-base-200 shadow">
                    <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">
                        Policy Details
                    </div>
                    <div className="h-full p-3">
                        {policyDetails.map(field => {
                            if (!fieldsMap[field]) return '';
                            const { label } = fieldsMap[field];
                            const value = fieldsMap[field].formatValue(policy[field], this.props);
                            if (!value || (Array.isArray(value) && !value.length)) return '';
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

    renderFieldsByPolicyCategories = () => {
        const policy = this.removeEmptyFields(this.props.policy);
        const policyCategoryFields = Object.keys(pick(policy, categories));
        return policyCategoryFields.map(category => {
            const policyCategoryLabel = categoryGroupsMap[category];
            const fields = Object.keys(policy[category]);
            if (!fields.length) return '';
            return (
                <div className="px-3 py-4 border-b border-base-300" key={category}>
                    <div className="bg-white border border-base-200 shadow">
                        <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">
                            {policyCategoryLabel}
                        </div>
                        <div className="h-full p-3">
                            {fields.map(field => {
                                if (!fieldsMap[field]) return '';
                                const { label } = fieldsMap[field];
                                const value = fieldsMap[field].formatValue(policy[category][field]);
                                if (!value || (Array.isArray(value) && !value.length)) return '';
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
        });
    };

    render() {
        const { policy } = this.props;
        if (!policy) return null;

        return (
            <div className="flex flex-1 flex-col bg-base-100">
                {this.renderFields()}
                {this.renderFieldsByPolicyCategories()}
            </div>
        );
    }
}

const getPolicyId = (state, props) => props.policyId;

const getPolicy = createSelector(
    [selectors.getPoliciesById, getPolicyId],
    (policiesById, policyId) => {
        const selectedPolicy = policiesById[policyId];
        if (!selectedPolicy) return null;
        return selectedPolicy;
    }
);

const mapStateToProps = createStructuredSelector({
    notifiers: selectors.getNotifiers,
    policy: getPolicy
});

export default withRouter(connect(mapStateToProps)(PolicyDetails));
