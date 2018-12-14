import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { components } from 'react-select';

import * as Icon from 'react-feather';
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
        onDelete: PropTypes.func.isRequired,
        id: PropTypes.string,
        toggleModal: PropTypes.func.isRequired
    };

    static defaultProps = {
        id: ''
    };

    renderMenuListComponent = props => <MenuList toggleModal={this.props.toggleModal} {...props} />;

    render() {
        const { fields, onDelete, id } = this.props;
        const { keyOptions, roleOptions } = formDescriptor.attrToRole;
        const addRule = () => fields.push({ props: { auth_provider_id: id } });
        const deleteRule = (group, idx) => () => {
            onDelete(group.get(idx));
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
                            />
                        </div>
                        <div className="w-full">
                            <Field jsonPath={`${group}.props.value`} type="text" label="Value" />
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
                            />
                        </div>
                        <button className="pl-2 pr-2" type="button">
                            <Icon.Plus className="h-4 w-4" />
                        </button>
                        <button className="pl-2 pr-2" type="button">
                            <Icon.Trash2 className="h-4 w-4" onClick={deleteRule(value, idx)} />
                        </button>
                    </div>
                ))}
                {/* eslint-disable-next-line */}
                <button className="btn btn-primary" type="button" onClick={addRule}>
                    Add New Rule
                </button>
            </div>
        );
    }
}

export default RuleGroups;
