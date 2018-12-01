import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/roles';

import List from 'Components/List';
import Panel, { headerClassName } from 'Components/Panel';

function SideBar({ header, onCreateNewRole, roles, selectedRole, selectRole, onCancel }) {
    const onRowSelectHandler = () => role => {
        selectRole(role);
        onCancel();
    };
    const panelHeaderClassName = `${headerClassName} bg-base-100`;
    return (
        <Panel header={header} headerClassName={panelHeaderClassName}>
            <div className="flex flex-col w-full h-full bg-base-100">
                <div className="overflow-auto">
                    <List
                        rows={roles}
                        selectRow={onRowSelectHandler()}
                        selectedListItem={selectedRole}
                        selectedIdAttribute="name"
                    />
                </div>
                <div className="flex items-center justify-center p-4 border-t border-base-300">
                    <div>
                        <button className="btn btn-primary" type="button" onClick={onCreateNewRole}>
                            Add New Role
                        </button>
                    </div>
                </div>
            </div>
        </Panel>
    );
}

SideBar.propTypes = {
    header: PropTypes.string.isRequired,
    roles: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    selectedRole: PropTypes.shape({}),
    selectRole: PropTypes.func.isRequired,
    onCreateNewRole: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired
};

SideBar.defaultProps = {
    selectedRole: null
};

const mapStateToProps = createStructuredSelector({
    roles: selectors.getRoles,
    selectedRole: selectors.getSelectedRole
});

const mapDispatchToProps = {
    selectRole: actions.selectRole
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SideBar);
