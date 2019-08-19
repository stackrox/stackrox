import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import merge from 'lodash/merge';

import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';

import PageHeader from 'Components/PageHeader';
import FormEditButtons from 'Components/FormEditButtons';
import Form from './Form';
import Detail from './Detail';

export const keyClassName = 'py-2 text-base-600 font-700 capitalize';
export const pageLayoutClassName = 'flex flex-col overflow-auto px-2 py-5 w-full';

const defaultPublicConfig = {
    header: {
        color: '#000000',
        backgroundColor: '#ffffff'
    },
    footer: {
        color: '#000000',
        backgroundColor: '#ffffff'
    },
    loginNotice: null
};

const defaultPrivateConfig = {
    alertConfig: {
        allRuntimeRetentionDurationDays: 30,
        deletedRuntimeRetentionDurationDays: 7,
        resolvedDeployRetentionDurationDays: 7
    },
    imageRetentionDurationDays: 7
};

function getPrivateConfig(privateConfig = {}) {
    return merge(defaultPrivateConfig, privateConfig);
}

const Page = ({ systemConfig, saveSystemConfig }) => {
    const [isEditing, setIsEditing] = useState(false);

    function saveHandler(config) {
        saveSystemConfig(config);
        setIsEditing(false);
    }

    function getContent() {
        const modifiedSystemConfig = { ...systemConfig };
        if (!systemConfig.publicConfig) modifiedSystemConfig.publicConfig = defaultPublicConfig;

        if (isEditing) {
            const safePrivateConfig = getPrivateConfig(systemConfig.privateConfig);
            const safeEditConfig = { ...modifiedSystemConfig, privateConfig: safePrivateConfig };

            return (
                <Form
                    initialValues={safeEditConfig}
                    onSubmit={saveHandler}
                    config={safeEditConfig}
                />
            );
        }
        return <Detail config={modifiedSystemConfig} />;
    }

    return (
        <section className="flex flex-1 flex-col h-full w-full">
            <div className="flex flex-1 flex-col w-full">
                <PageHeader header="System Configuration">
                    <div className="flex flex-1 justify-end">
                        <FormEditButtons
                            formName="system-config-form"
                            isEditing={isEditing}
                            setIsEditing={setIsEditing}
                        />
                    </div>
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
        publicConfig: defaultPublicConfig
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
