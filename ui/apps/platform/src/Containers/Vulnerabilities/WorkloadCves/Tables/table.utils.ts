import { gql } from '@apollo/client';
import sortBy from 'lodash/sortBy';
import { severityRankings } from 'constants/vulnerabilities';
import { VulnerabilitySeverity, isVulnerabilitySeverity } from 'types/cve.proto';
import { ApiSortOption } from 'types/search';

export type ImageMetadataContext = {
    id: string;
    name: {
        registry: string;
        remote: string;
        tag: string;
    } | null;
    metadata: {
        v1: {
            layers: {
                instruction: string;
                value: string;
            }[];
        } | null;
    } | null;
};

export const imageMetadataContextFragment = gql`
    fragment ImageMetadataContext on Image {
        id
        name {
            registry
            remote
            tag
        }
        metadata {
            v1 {
                layers {
                    instruction
                    value
                }
            }
        }
    }
`;

// This is a general type that isn't specific to this component, so should be moved elsewhere if
// there is a more appropriate location. (top level ./types/ directory?)
export type SourceType =
    | 'OS'
    | 'PYTHON'
    | 'JAVA'
    | 'RUBY'
    | 'NODEJS'
    | 'DOTNETCORERUNTIME'
    | 'INFRASTRUCTURE';

// TODO Enforce a non-empty imageVulnerabilities array at a higher level?
export type ComponentVulnerabilityBase = {
    type: 'Image' | 'Deployment';
    name: string;
    version: string;
    location: string;
    source: SourceType;
    layerIndex: number | null;
    imageVulnerabilities: {
        vulnerabilityId: string;
        severity: string;
        fixedByVersion: string;
        pendingExceptionCount: number;
    }[];
};

export type ImageComponentVulnerability = ComponentVulnerabilityBase;

export type DeploymentComponentVulnerability = Omit<
    ComponentVulnerabilityBase,
    'imageVulnerabilities'
> & {
    imageVulnerabilities: {
        vulnerabilityId: string;
        severity: string;
        cvss: number;
        scoreVersion: string;
        fixedByVersion: string;
        discoveredAtImage: string | null;
        pendingExceptionCount: number;
    }[];
};

export type TableDataRow = {
    image: {
        id: string;
        name: {
            remote: string;
            registry: string;
            tag: string;
        } | null;
    };
    name: string;
    vulnerabilityId: string;
    fixedByVersion: string;
    severity: VulnerabilitySeverity;
    version: string;
    location: string;
    source: SourceType;
    layer: {
        line: number;
        instruction: string;
        value: string;
    } | null;
    pendingExceptionCount: number;
};

/**
 * Given an image and its nested components and vulnerabilities, flatten the data into a single
 * level for display in a table. Note that this function assumes that the vulnerabilities array
 * for each component only has one element, which is the case when the query is filtered by CVE ID.
 *
 * @param imageMetadataContext The image context to use for the table rows
 * @param componentVulnerabilities The nested component -> vulnerabilities data for the image
 *
 * @returns The flattened table data
 */
export function flattenImageComponentVulns(
    imageMetadataContext: ImageMetadataContext,
    componentVulnerabilities: ImageComponentVulnerability[]
): TableDataRow[] {
    const image = imageMetadataContext;
    const layers = imageMetadataContext.metadata?.v1?.layers ?? [];

    return componentVulnerabilities.map((component) => {
        const vulnerability = component.imageVulnerabilities[0];
        return extractCommonComponentFields(image, layers, component, vulnerability);
    });
}

export function flattenDeploymentComponentVulns(
    imageMetadataContext: ImageMetadataContext,
    componentVulnerabilities: DeploymentComponentVulnerability[]
): (TableDataRow & {
    cvss: number;
    scoreVersion: string;
})[] {
    const image = imageMetadataContext;
    const layers = imageMetadataContext.metadata?.v1?.layers ?? [];

    return componentVulnerabilities.map((component) => {
        const vulnerability = component.imageVulnerabilities[0];
        const cvss = vulnerability?.cvss ?? 0;
        const scoreVersion = vulnerability?.scoreVersion ?? 'N/A';

        return {
            ...extractCommonComponentFields(image, layers, component, vulnerability),
            scoreVersion,
            cvss,
        };
    });
}

function extractCommonComponentFields(
    image: ImageMetadataContext,
    layers: { instruction: string; value: string }[],
    component: ComponentVulnerabilityBase,
    vulnerability: ComponentVulnerabilityBase['imageVulnerabilities'][0] | undefined
): TableDataRow {
    const { name, version, location, source, layerIndex } = component;

    let layer: TableDataRow['layer'] = null;

    if (layerIndex !== null) {
        const targetLayer = layers[layerIndex];
        if (targetLayer) {
            layer = {
                line: layerIndex + 1,
                instruction: targetLayer.instruction,
                value: targetLayer.value,
            };
        }
    }

    const vulnerabilityId = vulnerability?.vulnerabilityId ?? 'N/A';
    const severity =
        vulnerability?.severity && isVulnerabilitySeverity(vulnerability.severity)
            ? vulnerability.severity
            : 'UNKNOWN_VULNERABILITY_SEVERITY';
    const fixedByVersion = vulnerability?.fixedByVersion ?? 'N/A';
    const pendingExceptionCount = vulnerability?.pendingExceptionCount ?? 0;

    return {
        name,
        version,
        location,
        source,
        image,
        layer,
        vulnerabilityId,
        severity,
        fixedByVersion,
        pendingExceptionCount,
    };
}

export function sortTableData<TableRowType extends TableDataRow>(
    tableData: TableRowType[],
    sortOption: ApiSortOption
): TableRowType[] {
    const sortedRows = sortBy(tableData, (row) => {
        switch (sortOption.field) {
            case 'Image':
                return row.image.name?.remote ?? '';
            case 'Component':
                return row.name;
            default:
                return '';
        }
    });

    if (sortOption.reversed) {
        sortedRows.reverse();
    }
    return sortedRows;
}

/**
 * Get the highest severity of any vulnerability in the image.
 */
export function getHighestVulnerabilitySeverity(
    imageComponents: ImageComponentVulnerability[]
): VulnerabilitySeverity {
    let topSeverity: VulnerabilitySeverity = 'UNKNOWN_VULNERABILITY_SEVERITY';
    imageComponents.forEach((component) => {
        component.imageVulnerabilities.forEach(({ severity }) => {
            if (
                isVulnerabilitySeverity(severity) &&
                severityRankings[severity] > severityRankings[topSeverity]
            ) {
                topSeverity = severity;
            }
        });
    });
    return topSeverity;
}

/**
 * Get whether or not the image has any fixable vulnerabilities.
 */
export function getAnyVulnerabilityIsFixable(
    imageComponents: ImageComponentVulnerability[]
): boolean {
    return imageComponents.some((component) =>
        component.imageVulnerabilities.some(({ fixedByVersion }) => fixedByVersion !== '')
    );
}

export function getHighestCvssScore(
    imageComponents: {
        imageVulnerabilities: {
            cvss: number;
            scoreVersion: string;
        }[];
    }[]
): {
    cvss: number;
    scoreVersion: string;
} {
    let topCvss = 0;
    let topScoreVersion = 'N/A';
    imageComponents.forEach((component) => {
        component.imageVulnerabilities.forEach(({ cvss, scoreVersion }) => {
            if (cvss > topCvss) {
                topCvss = cvss;
                topScoreVersion = scoreVersion;
            }
        });
    });
    return { cvss: topCvss, scoreVersion: topScoreVersion };
}
