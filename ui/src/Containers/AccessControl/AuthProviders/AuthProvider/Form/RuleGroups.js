import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { formValues } from 'redux-form';
import uniqBy from 'lodash/uniqBy';

import { components } from 'react-select';
import * as Icon from 'react-feather';
import { selectMenuOnTopStyles } from 'Components/ReactSelect';
import Field from './Field';
import formDescriptor from './formDescriptor';

const MenuList = ({ toggleModal, ...props }) => (
    <components.MenuList {...props}>
        {props.children}
        <div className="border-b border-primary-400 w-full p-1">
            <button
                type="button"
                className="border border-primary-600 text-primary-600 p-3 w-full"
                onClick={toggleModal}
            >
                Create New Role
            </button>
        </div>
    </components.MenuList>
);

class RuleGroups extends Component {
    static propTypes = {
        fields: PropTypes.shape({}).isRequired,
        toggleModal: PropTypes.func.isRequired,
        usersAttributes: PropTypes.arrayOf(
            PropTypes.shape({
                authProviderId: PropTypes.string,
                key: PropTypes.string,
                value: PropTypes.string
            })
        ).isRequired
    };

    static defaultProps = {
        initialValues: {
            id: ''
        }
    };

    renderMenuListComponent = props => <MenuList toggleModal={this.props.toggleModal} {...props} />;

    getFilteredValueOptions = (valueOptions, idx) => {
        const { key } = this.props.groups[idx].props;
        const result = valueOptions
            .filter(option => option.key === key)
            .map(option => ({ label: option.label, value: option.value }));
        return result;
    };

    render() {
        const { fields, initialValues, usersAttributes } = this.props;
        const { keyOptions, roleOptions } = formDescriptor.attrToRole;
        let valueOptions = initialValues.groups.map(({ props: { key, value } }) => ({
            key,
            label: value,
            value
        }));
        valueOptions = uniqBy(
            usersAttributes
                .map(({ key, value }) => ({
                    key,
                    label: value,
                    value
                }))
                .concat(valueOptions),
            'value'
        );
        const addRule = () => fields.push({ props: { auth_provider_id: initialValues.id } });
        const deleteRule = (group, idx) => () => {
            fields.remove(idx);
        };
        return (
            <div className="w-full p-2">
                {fields.map((group, idx, value) => (
                    <div className="flex flex-row" key={idx}>
                        <div className="w-full">
                            <Field
                                jsonPath={`${group}.props.key`}
                                type="select"
                                label="Key"
                                options={keyOptions}
                                styles={selectMenuOnTopStyles}
                            />
                        </div>
                        <div className="w-full">
                            <Field
                                jsonPath={`${group}.props.value`}
                                type="selectcreatable"
                                label="Value"
                                options={this.getFilteredValueOptions(valueOptions, idx)}
                                styles={selectMenuOnTopStyles}
                            />
                        </div>
                        <div className="flex items-center">
                            <Icon.ArrowRight className="h-4 w-4" />
                        </div>
                        <div className="w-full">
                            <Field
                                jsonPath={`${group}.roleName`}
                                type="select"
                                label="Role"
                                options={roleOptions}
                                customComponents={{
                                    MenuList: this.renderMenuListComponent
                                }}
                                styles={selectMenuOnTopStyles}
                            />
                        </div>
                        <button className="pl-2 pr-2" type="button">
                            <Icon.Trash2 className="h-4 w-4" onClick={deleteRule(value, idx)} />
                        </button>
                    </div>
                ))}
                {/* eslint-disable-next-line */}
                <button className="btn btn-primary ml-1" type="button" onClick={addRule}>
                    Add New Rule
                </button>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    usersAttributes: selectors.getUsersAttributes
});

const mapDispatchToProps = {};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(formValues('groups')(RuleGroups));
