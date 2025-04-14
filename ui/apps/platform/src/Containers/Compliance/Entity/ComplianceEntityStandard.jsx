import React, { useState } from 'react';

import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import ComplianceList from 'Containers/Compliance/List/List';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import ComplianceSearchInput from '../ComplianceSearchInput';
import ListHeader from '../List/ListHeader';

const ComplianceEntityStandard = ({
    entityId,
    listEntityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
}) => {
    const [isExporting, setIsExporting] = useState(false);
    const listQuery = {
        'Standard Id': entityId,
    };
    return (
        <section className="flex flex-col h-full relative" id="capture-list">
            <ListHeader
                entityType="CONTROL"
                searchComponent={
                    <ComplianceSearchInput
                        placeholder="Filter standards"
                        categories={['COMPLIANCE']}
                        shouldAddComplianceState
                    />
                }
                isExporting={isExporting}
                setIsExporting={setIsExporting}
            />
            <ComplianceList
                entityType={listEntityType1}
                query={listQuery}
                selectedRowId={entityId1}
                entityType2={entityType2}
                entityListType2={entityListType2}
                entityId2={entityId2}
            />
            {isExporting && <BackdropExporting />}
        </section>
    );
};

ComplianceEntityStandard.propTypes = entityPagePropTypes;
ComplianceEntityStandard.defaultProps = entityPageDefaultProps;

export default ComplianceEntityStandard;
