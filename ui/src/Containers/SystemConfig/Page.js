import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';

import PageHeader from 'Components/PageHeader';
import SaveButton from 'Components/SaveButton';
import Form from './Form';
import Detail from './Detail';

export const keyClassName = 'py-2 text-base-600 font-700 capitalize';
export const pageLayoutClassName = 'flex flex-col overflow-auto px-2 py-5 w-full';

const Page = ({ systemConfig, saveSystemConfig }) => {
    const [isEditing, setIsEditing] = useState(false);

    function setEditingTrue() {
        setIsEditing(true);
    }

    function setEditingFalse() {
        setIsEditing(false);
    }

    function saveHandler(config) {
        saveSystemConfig(config);
        setIsEditing(false);
    }

    function getHeaderButtons() {
        if (isEditing) {
            return (
                <>
                    <button
                        className="btn btn-base mr-2"
                        type="button"
                        onClick={setEditingFalse}
                        data-test-id="cancel-btn"
                    >
                        Cancel
                    </button>
                    <SaveButton formName="system-config-form" />
                </>
            );
        }
        return (
            <button
                data-test-id="edit-btn"
                className="btn btn-base"
                type="button"
                onClick={setEditingTrue}
                disabled={isEditing}
            >
                Edit
            </button>
        );
    }

    function getContent() {
        if (isEditing) {
            return <Form initialValues={systemConfig} onSubmit={saveHandler} />;
        }
        return <Detail config={systemConfig} />;
    }

    return (
        <section className="flex flex-1 flex-col h-full w-full">
            <div className="flex flex-1 flex-col w-full">
                <PageHeader header="System Configuration">
                    <div className="flex flex-1 justify-end">{getHeaderButtons()}</div>
                </PageHeader>
                <div className="w-full h-full flex pb-0 bg-base-200 overflow-auto">
                    {getContent()}
                </div>
            </div>
        </section>
    );
};

Page.propTypes = {
    systemConfig: PropTypes.shape({}),
    saveSystemConfig: PropTypes.func.isRequired
};

Page.defaultProps = {
    systemConfig: {
        publicConfig: {
            header: null,
            footer: null,
            loginNotice: null
        }
    }
};

const mapStateToProps = createStructuredSelector({
    systemConfig: selectors.getSystemConfig
});

const mapDispatchToProps = {
    saveSystemConfig: actions.saveSystemConfig
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Page);
