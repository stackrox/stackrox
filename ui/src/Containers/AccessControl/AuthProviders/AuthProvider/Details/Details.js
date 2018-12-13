import React from 'react';
import PropTypes from 'prop-types';
import get from 'lodash/get';

import CollapsibleCard from 'Components/CollapsibleCard';
import formDescriptor from '../Form/formDescriptor';

const Field = props => {
    const { label, jsonPath, authProvider } = props;
    const value = get(authProvider, jsonPath);
    if (!value) return null;
    return (
        <div className="mb-4">
            <div className="py-2 text-base-600 font-700">{label}</div>
            <div className="w-1/2">{get(authProvider, jsonPath)}</div>
        </div>
    );
};

const Details = props => {
    const { name, type } = props.authProvider;
    const { groups } = props;

    if (!name) return null;
    const title = `1. ${name} Configuration`;
    const propsTitle = `2. Assign StackRox Roles to your ${name} address`;
    const fields = formDescriptor[type];
    return (
        <div className="w-full justify-between overflow-auto">
            <CollapsibleCard title={title}>
                <div className="w-full h-full p-4">
                    {fields &&
                        fields.map((field, index) => (
                            <Field key={index} {...field} authProvider={props.authProvider} />
                        ))}
                </div>
            </CollapsibleCard>
            <div className="mt-4">
                <CollapsibleCard title={propsTitle}>
                    <div className="flex flex-col">
                        {groups.map((group, idx) => (
                            <div className="p-4 flex flex-row w-full" key={idx}>
                                <div className="w-full">
                                    <div className="text-base-600 font-700 pb-2">Key</div>
                                    <div>{group.props.key}</div>
                                </div>
                                <div className="w-full">
                                    <div className="text-base-600 font-700 pb-2">Value</div>
                                    <div>{group.props.value}</div>
                                </div>
                                <div className="w-full">
                                    <div className="text-base-600 font-700 pb-2">Role</div>
                                    <div>{group.roleName}</div>
                                </div>
                            </div>
                        ))}
                    </div>
                </CollapsibleCard>
            </div>
        </div>
    );
};

Field.propTypes = {
    label: PropTypes.string,
    jsonPath: PropTypes.string,
    authProvider: PropTypes.shape({
        name: PropTypes.string
    }).isRequired
};

Field.defaultProps = {
    label: '',
    jsonPath: ''
};

Details.propTypes = {
    authProvider: PropTypes.shape({
        name: PropTypes.string
    }).isRequired,
    groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

export default Details;
