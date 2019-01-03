import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/auth';

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
        this.props.setAuthProviderEditingState(false);
    };

    onDelete = authProvider => {
        this.props.deleteAuthProvider(authProvider.id);
        this.props.setAuthProviderEditingState(false);
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
        return (
            <section className="flex flex-1 h-full">
                <div className="w-1/4 m-4">{this.renderSideBar()}</div>
                <div className="w-3/4 my-4 mr-4">
                    <AuthProvider
                        isEditing={this.props.isEditing}
                        selectedAuthProvider={selectedAuthProvider}
                        onSave={this.onSave}
                        onEdit={this.onEdit}
                        onCancel={this.onCancel}
                        groups={groups}
                    />
                </div>
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
