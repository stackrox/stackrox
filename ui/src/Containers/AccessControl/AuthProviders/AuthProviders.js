import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/auth';
import Dialog from 'Components/Dialog';

import SideBar from 'Containers/AccessControl/SideBar';
import Select from 'Containers/AccessControl/AuthProviders/Select';
import AuthProvider from 'Containers/AccessControl/AuthProviders/AuthProvider/AuthProvider';

const AuthProviders = ({
    saveAuthProvider,
    setAuthProviderEditingState,
    selectAuthProvider,
    selectedAuthProvider,
    authProviders,
    deleteAuthProvider,
    groups,
    isEditing
}) => {
    const [providerToDelete, setProviderToDelete] = useState(null);

    function onEdit() {
        setAuthProviderEditingState(true);
    }

    function onCreateNewAuthProvider(option) {
        selectAuthProvider({ type: option.value });
        setAuthProviderEditingState(true);
    }

    function onCancel() {
        setAuthProviderEditingState(false);
        if (selectedAuthProvider && !selectedAuthProvider.id) {
            selectAuthProvider(authProviders[0]);
        }
    }

    function onDelete(authProvider) {
        setProviderToDelete(authProvider);
    }

    function deleteProvider() {
        const providerId = providerToDelete && providerToDelete.id;
        if (!providerId) return;

        deleteAuthProvider(providerId);
        setAuthProviderEditingState(false);
        setProviderToDelete(null);
    }

    function onCancelDeleteProvider() {
        setProviderToDelete(null);
    }

    const curProviderToDelete = providerToDelete && providerToDelete.name;

    const className = isEditing
        ? 'before before:absolute before:h-full before:opacity-50 before:bg-base-400 before:w-full before:z-10'
        : '';
    return (
        <section className="flex flex-1 h-full">
            <div className={`w-1/4 flex flex-col ${className}`}>
                <div className="m-4 h-full">
                    <SideBar
                        header="Auth Providers"
                        rows={authProviders}
                        selected={selectedAuthProvider}
                        onSelectRow={selectAuthProvider}
                        addRowButton={<Select onChange={onCreateNewAuthProvider} />}
                        onCancel={onCancel}
                        onDelete={onDelete}
                        type="auth provider"
                    />
                </div>
            </div>
            <div className="w-3/4 my-4 mr-4 z-10">
                <AuthProvider
                    isEditing={isEditing}
                    selectedAuthProvider={selectedAuthProvider}
                    onSave={saveAuthProvider}
                    onEdit={onEdit}
                    onCancel={onCancel}
                    groups={groups}
                />
            </div>
            <Dialog
                isOpen={!!curProviderToDelete}
                text={`Deleting "${curProviderToDelete}" will cause users to be logged out. Are you sure you want to delete "${curProviderToDelete}"?`}
                onConfirm={deleteProvider}
                onCancel={onCancelDeleteProvider}
                confirmText="Delete"
            />
        </section>
    );
};

AuthProviders.propTypes = {
    authProviders: PropTypes.arrayOf(PropTypes.shape({})),
    selectedAuthProvider: PropTypes.shape({}),
    selectAuthProvider: PropTypes.func.isRequired,
    saveAuthProvider: PropTypes.func.isRequired,
    deleteAuthProvider: PropTypes.func.isRequired,
    groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    setAuthProviderEditingState: PropTypes.func.isRequired,
    isEditing: PropTypes.bool
};

AuthProviders.defaultProps = {
    authProviders: [],
    selectedAuthProvider: null,
    isEditing: false
};

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAvailableAuthProviders,
    selectedAuthProvider: selectors.getSelectedAuthProvider,
    groups: selectors.getRuleGroups,
    isEditing: selectors.getAuthProviderEditingState
});

const mapDispatchToProps = {
    selectAuthProvider: actions.selectAuthProvider,
    saveAuthProvider: actions.saveAuthProvider,
    deleteAuthProvider: actions.deleteAuthProvider,
    setAuthProviderEditingState: actions.setAuthProviderEditingState
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(AuthProviders);
