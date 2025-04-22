import React from 'react';
import PropTypes from 'prop-types';
import upperFirst from 'lodash/upperFirst';
import startCase from 'lodash/startCase';

import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import useEntityName from 'hooks/useEntityName';
import entityLabels from 'messages/entity';
import { getConfigurationManagementEntityTypes } from 'utils/entityRelationships';

const EntityPageHeader = ({ entityType, entityId, urlParams, isExporting, setIsExporting }) => {
    const safeEntityId = decodeURIComponent(entityId); // fix bug  ROX-4543-fix-bad-encoding-in-config-mgt-API-request
    const { entityName } = useEntityName(entityType, safeEntityId);

    const header = entityName || '-';
    const subHeader = upperFirst(entityLabels[entityType]);
    const exportFilename = `${startCase(subHeader)} Report: "${header}"`;

    let pdfId = 'capture-dashboard-stretch';
    if (urlParams && urlParams.entityListType1) {
        pdfId = 'capture-list';
    }
    return (
        <PageHeader
            header={header}
            subHeader={subHeader}
            classes="z-1 pr-0 ignore-react-onclickoutside"
        >
            <div className="flex flex-1 justify-end h-full">
                <div className="flex items-center">
                    <ExportButton
                        fileName={exportFilename}
                        type={entityType}
                        page="configManagement"
                        pdfId={pdfId}
                        isExporting={isExporting}
                        setIsExporting={setIsExporting}
                    />
                </div>
                <div className="flex items-center pl-2">
                    <EntitiesMenu
                        text="All Entities"
                        options={getConfigurationManagementEntityTypes()}
                    />
                </div>
            </div>
        </PageHeader>
    );
};

EntityPageHeader.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    urlParams: PropTypes.shape({
        entityListType1: PropTypes.string,
    }),
    isExporting: PropTypes.bool.isRequired,
    setIsExporting: PropTypes.func.isRequired,
};

EntityPageHeader.defaultProps = {
    urlParams: null,
};

export default EntityPageHeader;
