import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/roles';

import SideBar from 'Containers/AccessControl/Roles/SideBar/SideBar';
import Permissions from 'Containers/AccessControl/Roles/Permissions/Permissions';

class Roles extends Component {
    static propTypes = {
        roles: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                globalAccess: PropTypes.string
            })
        ).isRequired,
        selectedRole: PropTypes.shape({
            name: PropTypes.string,
            globalAccess: PropTypes.string
        }),
        selectRole: PropTypes.func.isRequired,
        saveRole: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedRole: null
    };

    constructor(props) {
        super(props);
        this.state = {
            isEditing: false
        };
    }

    onSave = data => {
        this.props.saveRole(data);
        this.setState({ isEditing: false });
    };

    onEdit = () => {
        this.setState({ isEditing: true });
    };

    onCreateNewRole = () => {
        this.props.selectRole({});
        this.setState({ isEditing: true });
    };

    onCancel = () => {
        this.setState({ isEditing: false });
    };

    renderSideBar = () => {
        const header = 'StackRox Roles';
        const { roles } = this.props;
        return (
            <SideBar
                header={header}
                rows={roles}
                onCreateNewRole={this.onCreateNewRole}
                onCancel={this.onCancel}
            />
        );
    };

    render() {
        const { selectedRole } = this.props;
        return (
            <section className="flex flex-1 h-full">
                <div className="w-1/4 m-4">{this.renderSideBar()}</div>
                <div className="w-3/4 my-4 mr-4">
                    <Permissions
                        isEditing={this.state.isEditing}
                        selectedRole={selectedRole}
                        onSave={this.onSave}
                        onEdit={this.onEdit}
                        onCancel={this.onCancel}
                    />
                </div>
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    roles: selectors.getRoles,
    selectedRole: selectors.getSelectedRole
});

const mapDispatchToProps = {
    selectRole: actions.selectRole,
    saveRole: actions.saveRole
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Roles);
