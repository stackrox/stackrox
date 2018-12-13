import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/auth';

import List from 'Components/List';
import Panel, { headerClassName } from 'Components/Panel';
import Select from 'Containers/AccessControl/AuthProviders/SideBar/Select/Select';

function SideBar({
    header,
    onCreateNewAuthProvider,
    authProviders,
    selectedAuthProvider,
    selectAuthProvider,
    onCancel
}) {
    const onRowSelectHandler = () => authProvider => {
        selectAuthProvider(authProvider);
        onCancel();
    };
    const panelHeaderClassName = `${headerClassName} bg-base-100`;
    return (
        <Panel header={header} headerClassName={panelHeaderClassName}>
            <div className="flex flex-col w-full h-full bg-base-100">
                <div className="overflow-auto">
                    <List
                        rows={authProviders}
                        selectRow={onRowSelectHandler()}
                        selectedListItem={selectedAuthProvider}
                        selectedIdAttribute="name"
                    />
                </div>
                <div className="flex items-center justify-center p-4 border-t border-base-300">
                    <div>
                        <Select onChange={onCreateNewAuthProvider} />
                    </div>
                </div>
            </div>
        </Panel>
    );
}

SideBar.propTypes = {
    header: PropTypes.string.isRequired,
    authProviders: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    selectedAuthProvider: PropTypes.shape({}),
    selectAuthProvider: PropTypes.func.isRequired,
    onCreateNewAuthProvider: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired
};

SideBar.defaultProps = {
    selectedAuthProvider: null
};

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAvailableAuthProviders,
    selectedAuthProvider: selectors.getSelectedAuthProvider
});

const mapDispatchToProps = {
    selectAuthProvider: actions.selectAuthProvider
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SideBar);
