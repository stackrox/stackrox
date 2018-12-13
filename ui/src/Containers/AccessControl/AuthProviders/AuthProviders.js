import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';

import SideBar from 'Containers/AccessControl/AuthProviders/SideBar/SideBar';
import AuthProvider from 'Containers/AccessControl/AuthProviders/AuthProvider/AuthProvider';

class AuthProviders extends Component {
    static propTypes = {
        selectedAuthProvider: PropTypes.shape({}),
        selectAuthProvider: PropTypes.func.isRequired,
        saveAuthProvider: PropTypes.func.isRequired,
        saveRuleGroup: PropTypes.func.isRequired,
        deleteRuleGroup: PropTypes.func.isRequired,
        groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired
    };

    static defaultProps = {
        selectedAuthProvider: null
    };

    constructor(props) {
        super(props);
        this.state = {
            isEditing: false
        };
    }

    onSave = data => {
        const { groups, ...remaining } = data;
        this.props.saveAuthProvider(remaining);
        this.props.saveRuleGroup(groups);
        this.setState({ isEditing: false });
    };

    onEdit = () => {
        this.setState({ isEditing: true });
    };

    onCreateNewAuthProvider = option => {
        this.props.selectAuthProvider({ type: option.value });
        this.setState({ isEditing: true });
    };

    onCancel = () => {
        this.setState({ isEditing: false });
    };

    renderSideBar = () => {
        const header = 'Auth Providers';
        return (
            <SideBar
                header={header}
                onCreateNewAuthProvider={this.onCreateNewAuthProvider}
                onCancel={this.onCancel}
            />
        );
    };

    deleteRuleGroup = group => {
        this.props.deleteRuleGroup(group);
    };

    render() {
        const { selectedAuthProvider, groups } = this.props;
        return (
            <section className="flex flex-1 h-full">
                <div className="w-1/4 m-4">{this.renderSideBar()}</div>
                <div className="w-3/4 my-4 mr-4">
                    <AuthProvider
                        isEditing={this.state.isEditing}
                        selectedAuthProvider={selectedAuthProvider}
                        onSave={this.onSave}
                        onEdit={this.onEdit}
                        onCancel={this.onCancel}
                        onDelete={this.deleteRuleGroup}
                        groups={groups}
                    />
                </div>
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    selectedAuthProvider: selectors.getSelectedAuthProvider,
    groups: selectors.getRuleGroups
});

const mapDispatchToProps = {
    selectAuthProvider: actions.selectAuthProvider,
    saveAuthProvider: actions.saveAuthProvider,
    saveRuleGroup: groupActions.saveRuleGroup,
    deleteRuleGroup: groupActions.deleteRuleGroup
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(AuthProviders);
