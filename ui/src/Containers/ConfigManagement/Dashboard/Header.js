import React from 'react';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
import PoliciesTile from './widgets/PoliciesTile';
import CISControlsTile from './widgets/CISControlsTile';
import AppMenu from './widgets/AppMenu';
import RBACMenu from './widgets/RBACMenu';

const ConfigManagementHeader = ({ classes, bgStyle }) => {
    return (
        <PageHeader
            classes={classes}
            bgStyle={bgStyle}
            header="Configuration Management"
            subHeader="Dashboard"
        >
            <div className="flex flex-1 justify-end">
                <PoliciesTile />
                <CISControlsTile />
                <AppMenu />
                <RBACMenu />
                <div className="self-center">
                    <ExportButton
                        fileName="Config Mangement Dashboard Report"
                        type={null}
                        page="configManagement"
                        pdfId="capture-dashboard"
                    />
                </div>
            </div>
        </PageHeader>
    );
};

ConfigManagementHeader.propTypes = {
    classes: PropTypes.string,
    bgStyle: PropTypes.shape({})
};

ConfigManagementHeader.defaultProps = {
    classes: null,
    bgStyle: null
};

export default ConfigManagementHeader;
