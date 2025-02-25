import { gql } from '@apollo/client';
import { min, parse } from 'date-fns';
import sortBy from 'lodash/sortBy';
import uniq from 'lodash/uniq';
import pluralize from 'pluralize';

import { CveBaseInfo, VulnerabilitySeverity, isVulnerabilitySeverity } from 'types/cve.proto';
import { SourceType } from 'types/image.proto';
import { ApiSortOptionSingle } from 'types/search';

import {
    getHighestVulnerabilitySeverity,
    getIsSomeVulnerabilityFixable,
} from '../../utils/vulnerabilityUtils';

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

// TODO Enforce a non-empty imageVulnerabilities array at a higher level?
export type ComponentVulnerabilityBase = {
    type: 'Image' | 'Deployment';
    name: string;
    version: string;
    location: string;
    source: SourceType;
    layerIndex: number | null;
    imageVulnerabilities: {
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
        severity: string;
        cvss: number;
        scoreVersion: string;
        fixedByVersion: string;
        discoveredAtImage: string | null;
        publishedOn: string | null;
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
        severity,
        fixedByVersion,
        pendingExceptionCount,
    };
}

export function sortTableData<TableRowType extends TableDataRow>(
    tableData: TableRowType[],
    sortOption: ApiSortOptionSingle
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

export type DeploymentWithVulnerabilities = {
    id: string;
    images: ImageMetadataContext[];
    imageVulnerabilities: {
        vulnerabilityId: string;
        cve: string;
        cveBaseInfo: CveBaseInfo;
        operatingSystem: string;
        publishedOn: string | null;
        summary: string;
        pendingExceptionCount: number;
        images: {
            imageId: string;
            imageComponents: DeploymentComponentVulnerability[];
        }[];
    }[];
};

type DeploymentVulnerabilityImageMapping = {
    imageMetadataContext: ImageMetadataContext;
    componentVulnerabilities: DeploymentComponentVulnerability[];
};

export type FormattedDeploymentVulnerability = {
    vulnerabilityId: string;
    cve: string;
    cveBaseInfo: CveBaseInfo;
    operatingSystem: string;
    severity: VulnerabilitySeverity;
    isFixable: boolean;
    discoveredAtImage: Date | null;
    publishedOn: Date | null;
    summary: string;
    affectedComponentsText: string;
    images: DeploymentVulnerabilityImageMapping[];
    pendingExceptionCount: number;
};

export function formatVulnerabilityData(
    deployment: DeploymentWithVulnerabilities
): FormattedDeploymentVulnerability[] {
    // Create a map of image ID to image metadata for easy lookup
    // We use 'Partial' here because there is no guarantee that the image will be found
    const imageMap: Partial<Record<string, ImageMetadataContext>> = {};
    deployment.images.forEach((image) => {
        imageMap[image.id] = image;
    });

    return deployment.imageVulnerabilities.map((vulnerability) => {
        const {
            vulnerabilityId,
            cve,
            cveBaseInfo,
            operatingSystem,
            summary,
            images,
            pendingExceptionCount,
        } = vulnerability;
        // Severity, Fixability, and Discovered date are all based on the aggregate value of all components
        const allVulnerableComponents = vulnerability.images.flatMap((img) => img.imageComponents);
        const allVulnerabilities = allVulnerableComponents.flatMap((c) => c.imageVulnerabilities);
        const highestVulnSeverity = getHighestVulnerabilitySeverity(allVulnerabilities);
        const isFixableInDeployment = getIsSomeVulnerabilityFixable(allVulnerabilities);
        const allDiscoveredDates = allVulnerableComponents
            .flatMap((c) => c.imageVulnerabilities.map((v) => v.discoveredAtImage))
            .filter((d): d is string => d !== null);
        const oldestDiscoveredVulnDate = min(...allDiscoveredDates);
        // TODO This logic is used in many places, could extract to a util
        const uniqueComponents = uniq(allVulnerableComponents.map((c) => c.name));
        const affectedComponentsText =
            uniqueComponents.length === 1
                ? uniqueComponents[0]
                : `${uniqueComponents.length} components`;

        const vulnerabilityImages = images
            .map((img) => ({
                imageMetadataContext: imageMap[img.imageId],
                componentVulnerabilities: img.imageComponents,
            }))
            // filter out values where the vulnerability->image mapping is missing
            .filter(
                (vulnImageMap): vulnImageMap is DeploymentVulnerabilityImageMapping =>
                    !!vulnImageMap.imageMetadataContext
            );

        const publishedOnDate = allVulnerabilities[0].publishedOn
            ? parse(allVulnerabilities[0].publishedOn)
            : null;

        return {
            vulnerabilityId,
            cve,
            cveBaseInfo,
            operatingSystem,
            severity: highestVulnSeverity,
            isFixable: isFixableInDeployment,
            discoveredAtImage: oldestDiscoveredVulnDate,
            publishedOn: publishedOnDate,
            summary,
            affectedComponentsText,
            images: vulnerabilityImages,
            pendingExceptionCount,
        };
    });
}

export function getCveBaseInfoFromDistroTuples(
    distroTuples: { cveBaseInfo: CveBaseInfo }[]
): CveBaseInfo | undefined {
    // Return cveBaseInfo that has max value of epssProbability,
    // consistent with aggregateFunc: 'max' property in sortUtils.tsx file.
    let cveBaseInfoMax: CveBaseInfo | undefined;

    if (Array.isArray(distroTuples)) {
        let epssProbabilityMax = -1; // in case epssProbability is ever zero
        distroTuples.forEach(({ cveBaseInfo }) => {
            if (cveBaseInfo?.epss && cveBaseInfo.epss?.epssProbability > epssProbabilityMax) {
                cveBaseInfoMax = cveBaseInfo;
                epssProbabilityMax = cveBaseInfo.epss.epssProbability;
            }
        });
    }

    return cveBaseInfoMax;
}

// Given probability as float fraction, return as percent with 3 decimal digits.
export function formatEpssProbabilityAsPercent(epssProbability: number | undefined) {
    if (typeof epssProbability === 'number' && epssProbability >= 0 && epssProbability <= 1) {
        const epssPercent = epssProbability * 100;
        return `${epssPercent.toFixed(3)}%`;
    }

    // For any of the following: null, undefined, or number out of range
    return 'Not available';
}

export function formatTotalAdvisories(totalAdvisories: number | undefined) {
    if (
        typeof totalAdvisories === 'number' &&
        Number.isSafeInteger(totalAdvisories) &&
        totalAdvisories > 0
    ) {
        return `${totalAdvisories} ${pluralize('advisory', totalAdvisories)}`;
    }

    // For any of the following: undefined, or number out of range
    return 'No advisories';
}
