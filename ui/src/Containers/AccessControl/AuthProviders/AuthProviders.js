import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/auth';
import Dialog from 'Components/Dialog';

import SideBar from 'Containers/AccessControl/SideBar';
import Select from 'Containers/AccessControl/AuthProviders/Select';
import AuthProvider from 'Containers/AccessControl/AuthProviders/AuthProvider/AuthProvider';

class AuthProviders extends Component {
    static propTypes = {
        authProviders: PropTypes.arrayOf(PropTypes.shape({})),
        selectedAuthProvider: PropTypes.shape({}),
        selectAuthProvider: PropTypes.func.isRequired,
        saveAuthProvider: PropTypes.func.isRequired,
        deleteAuthProvider: PropTypes.func.isRequired,
        groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        setAuthProviderEditingState: PropTypes.func.isRequired,
        isEditing: PropTypes.bool
    };

    static defaultProps = {
        authProviders: [],
        selectedAuthProvider: null,
        isEditing: false
    };

    constructor(props) {
        super(props);
        this.state = {
            providerToDelete: null
        };
    }

    onSave = data => {
        const { saveAuthProvider } = this.props;
        saveAuthProvider(data);
    };

    onEdit = () => {
        this.props.setAuthProviderEditingState(true);
    };

    onCreateNewAuthProvider = option => {
        this.props.selectAuthProvider({ type: option.value });
        this.props.setAuthProviderEditingState(true);
    };

    onCancel = () => {
        const {
            selectedAuthProvider,
            setAuthProviderEditingState,
            selectAuthProvider,
            authProviders
        } = this.props;
        setAuthProviderEditingState(false);
        if (selectedAuthProvider && !selectedAuthProvider.id) {
            selectAuthProvider(authProviders[0]);
        }
    };

    onDelete = authProvider => {
        this.setState({
            providerToDelete: authProvider
        });
    };

    deleteProvider = () => {
        const providerId = this.state.providerToDelete && this.state.providerToDelete.id;
        if (!providerId) return;

        this.props.deleteAuthProvider(providerId);
        this.props.setAuthProviderEditingState(false);
        this.setState({
            providerToDelete: null
        });
    };

    onCancelDeleteProvider = () => {
        this.setState({
            providerToDelete: null
        });
    };

    renderSideBar = () => {
        const header = 'Auth Providers';
        const { authProviders, selectedAuthProvider, selectAuthProvider } = this.props;
        return (
            <SideBar
                header={header}
                rows={authProviders}
                selected={selectedAuthProvider}
                onSelectRow={selectAuthProvider}
                addRowButton={<Select onChange={this.onCreateNewAuthProvider} />}
                onCancel={this.onCancel}
                onDelete={this.onDelete}
                type="auth provider"
            />
        );
    };

    render() {
        const { selectedAuthProvider, groups } = this.props;
        const providerToDelete = this.state.providerToDelete && this.state.providerToDelete.name;

        const className = this.props.isEditing
            ? 'before before:absolute before:h-full before:opacity-50 before:bg-secondary-900 before:w-full before:z-10'
            : '';
        return (
            <section className="flex flex-1 h-full">
                <div className={`w-1/4 flex flex-col ${className}`}>
                    <div className="m-4 h-full">{this.renderSideBar()}</div>
                </div>
                <div className="w-3/4 my-4 mr-4 z-10">
                    <AuthProvider
                        isEditing={this.props.isEditing}
                        selectedAuthProvider={selectedAuthProvider}
                        onSave={this.onSave}
                        onEdit={this.onEdit}
                        onCancel={this.onCancel}
                        groups={groups}
                    />
                </div>
                <Dialog
                    isOpen={!!providerToDelete}
                    text={`Deleting "${providerToDelete}" will cause users to be logged out. Are you sure you want to delete "${providerToDelete}"?`}
                    onConfirm={this.deleteProvider}
                    onCancel={this.onCancelDeleteProvider}
                    confirmText="Delete"
                />
            </section>
        );
    }
}

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
