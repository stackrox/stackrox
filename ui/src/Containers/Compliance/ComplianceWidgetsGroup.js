import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';

const standardsMap = {
    [entityTypes.NODE]: [
        entityTypes.NIST_800_190,
        entityTypes.CIS_Kubernetes_v1_2_0,
        entityTypes.CIS_Docker_v1_1_0
    ],
    [entityTypes.DEPLOYMENT]: [
        entityTypes.PCI_DSS_3_2,
        entityTypes.NIST_800_190,
        entityTypes.HIPAA_164
    ]
};

const ComplianceWidgetsGroup = ({
    controlResult,
    entityType,
    entityName,
    entityId,
    pdfClassName
}) => {
    if (controlResult) return null;
    const complianceByStandardWidgets = standardsMap[entityType].map(standard => (
        <ComplianceByStandard
            key={standard}
            standardType={standard}
            entityName={entityName}
            entityId={entityId}
            entityType={entityType}
            className={pdfClassName}
        />
    ));
    return complianceByStandardWidgets;
};

ComplianceWidgetsGroup.propTypes = {
    controlResult: PropTypes.shape({}),
    entityType: PropTypes.string.isRequired,
    entityName: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    pdfClassName: PropTypes.string.isRequired
};

export default ComplianceWidgetsGroup;
