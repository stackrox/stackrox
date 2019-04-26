import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';

import { actions } from 'reducers/roles';
import Modal from 'Components/Modal';
import Form from 'Containers/AccessControl/Roles/Permissions/Form';
import SaveButton from 'Containers/AccessControl/SaveButton';
import { defaultNewRolePermissions } from 'constants/accessControl';

class CreateRoleModal extends Component {
    static propTypes = {
        saveRole: PropTypes.func.isRequired,
        onClose: PropTypes.func.isRequired
    };

    saveRoleHandler = data => {
        this.props.saveRole(data);
        this.props.onClose();
    };

    render() {
        return (
            <Modal isOpen onRequestClose={this.props.onClose} className="w-full lg:w-2/3">
                <header className="flex items-center w-full p-4 bg-primary-500 text-base-100 uppercase">
                    <span className="flex flex-1 uppercase">New Authorization Role</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.props.onClose} />
                </header>
                <div className="flex flex-1">
                    <div className="flex flex-col w-full">
                        <div className="border-b border-base-300 flex flex-col">
                            <Form
                                onSubmit={this.saveRoleHandler}
                                initialValues={{ resourceToAccess: defaultNewRolePermissions }}
                            />
                        </div>
                        <SaveButton className="min-h-10 w-1/4 mx-auto m-3" formName="role-form" />
                    </div>
                </div>
            </Modal>
        );
    }
}

const mapDispatchToProps = {
    saveRole: actions.saveRole
};

export default connect(
    null,
    mapDispatchToProps
)(CreateRoleModal);
