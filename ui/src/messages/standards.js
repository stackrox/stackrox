import { standardTypes } from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import capitalize from 'lodash/capitalize';

export const standardLabels = {
    [standardTypes.PCI_DSS_3_2]: 'PCI DSS 3.2.1',
    [standardTypes.NIST_800_190]: 'NIST 800-190',
    [standardTypes.HIPAA_164]: 'HIPAA 164',
    [standardTypes.CIS_Kubernetes_v1_2_0]: 'CIS Kubernetes v1.2.0',
    [standardTypes.CIS_Docker_v1_1_0]: 'CIS Docker v1.1.0'
};

export const standardShortLabels = {
    ...standardLabels,
    [standardTypes.CIS_Kubernetes_v1_2_0]: 'CIS K8s v1.2.0'
};

export const getStandardAcrossEntityLabel = (
    standardType,
    entityType,
    grammaticalNumberCategory
) => {
    return `${standardLabels[standardType]} Across ${pluralize(
        capitalize(entityLabels[entityType]),
        grammaticalNumberCategory === 'plural'
    )}`;
};

export default standardLabels;
