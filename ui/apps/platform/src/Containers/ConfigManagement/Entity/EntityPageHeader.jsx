import PropTypes from 'prop-types';
import upperFirst from 'lodash/upperFirst';

import PageHeader from 'Components/PageHeader';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import useEntityName from 'hooks/useEntityName';
import entityLabels from 'messages/entity';
import { getConfigurationManagementEntityTypes } from 'utils/entityRelationships';

const EntityPageHeader = ({ entityType, entityId }) => {
    const safeEntityId = decodeURIComponent(entityId); // fix bug  ROX-4543-fix-bad-encoding-in-config-mgt-API-request
    const { entityName } = useEntityName(entityType, safeEntityId);

    const header = entityName || '-';
    const subHeader = upperFirst(entityLabels[entityType]);

    return (
        <PageHeader
            header={header}
            subHeader={subHeader}
            classes="z-1 pr-0 ignore-react-onclickoutside"
        >
            <div className="flex flex-1 justify-end h-full">
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
};

export default EntityPageHeader;
