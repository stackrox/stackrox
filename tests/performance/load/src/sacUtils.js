import http from 'k6/http';
import { randomSeed } from 'k6';

const libTags = {
    lib: 'true',
};

function getRandomInt(max) {
    return Math.floor(Math.random() * max);
}

function shuffleStringArray(input) {
    const output = [];
    input.forEach(value => output.push(value));
    const l = output.length;
    for (let i = 0; i < l; i++) {
        const ix = getRandomInt(l-i-1);
        const valix = output[i + ix];
        const vali = output[i];
        output[i] = valix;
        output[ix] = vali;
    }
    return output;
}

// Fetch clusters and all namespaces for them sorted by cluster and namespace.
function getNamespacesByCluster(host, headers) {
    // We are using GraphQL here, because we can ensure order and less data will be returned.
    const response = http.post(
        `${host}/api/graphql`,
        '{"query":"{clusters(pagination:{sortOption:{field:\\"Cluster\\",reversed:false}}){namespaces(pagination:{sortOption:{field:\\"Namespace\\",reversed:false}}){metadata{clusterName,name}}}}"}',
        {
            headers,
            tags: libTags,
        }
    );

    const clusters = response.json('data.clusters');

    const mapClusterNamespaces = {};
    clusters.forEach(({ namespaces }) => {
        namespaces.forEach((namespace) => {
            if (!(namespace.metadata.clusterName in mapClusterNamespaces)) {
                mapClusterNamespaces[namespace.metadata.clusterName] = [];
            }

            mapClusterNamespaces[namespace.metadata.clusterName].push(namespace.metadata.name);
        });
    });

    return mapClusterNamespaces;
}

// Get randomized list of namespaces for access scope.
function getScopeNamespaces(namespacesByClusterMap, numClusters, numNamespacesPerCluster) {
    const clusterNames = Object.keys(namespacesByClusterMap);
    const shuffledClusterNames = shuffleStringArray(clusterNames);

    const usedClusters = new Set();
    const usedNamespaces = new Set();

    const includeNamespaces = [];
    let clusterIx = 0;
    while (usedClusters.size < numClusters && usedClusters.size < clusterNames.length) {
        // const clusterIndex = getRandomInt(clusterNames.length);
        // const clusterName = clusterNames[clusterIndex];
        const clusterName = shuffledClusterNames[clusterIx];
        clusterIx += 1;

        if (!usedClusters.has(clusterName)) {
            usedClusters.add(clusterName);

            const shuffledClusterNamespaces = shuffleStringArray(namespacesByClusterMap[clusterName]);

            let namespacesCount = 0;
            let namespaceIx = 0;
            while (
                namespacesCount < numNamespacesPerCluster &&
                // namespacesCount < namespacesByClusterMap[clusterName].length
                namespacesCount < shuffledClusterNamespaces.length
            ) {
                // const namespaceIndex = getRandomInt(namespacesByClusterMap[clusterName].length);
                // const namespaceName = namespacesByClusterMap[clusterName][namespaceIndex];
                const namespaceName = shuffledClusterNamespaces[namespaceIx];
                namespaceIx++;

                if (!usedNamespaces.has(`${clusterName}:${namespaceName}`)) {
                    namespacesCount += 1;
                    usedNamespaces.add(`${clusterName}:${namespaceName}`);

                    includeNamespaces.push({
                        clusterName: clusterName,
                        namespaceName: namespaceName,
                    });
                }
            }
        }
    }

    return includeNamespaces;
}

// Create access scope.
function createAccessScopeWithNamespaces(host, headers, name, includeNamespaces) {
    console.info('Create SAC', name, includeNamespaces);

    const requestPayload = {
        name: name,
        description: name,
        rules: {
            includedNamespaces: includeNamespaces,
            clusterLabelSelectors: [],
            namespaceLabelSelectors: [],
        },
    };

    const response = http
        .post(`${host}/v1/simpleaccessscopes`, JSON.stringify(requestPayload), {
            headers,
            tags: libTags,
        })
        .json();

    if (response['error']) {
        console.error('Error: createAccessScopeWithNamespaces', {
            requestPayload,
            response,
        });
    }

    return response;
}

// Create role for already existing access scope.
function createRole(host, headers, name, accessScopeId) {
    const requestPayload = {
        name: name,
        resourceToAccess: {},
        description: `Test: ${name}`,
        permissionSetId: 'ffffffff-ffff-fff4-f5ff-fffffffffffe',
        accessScopeId: accessScopeId,
    };

    const response = http
        .post(`${host}/v1/roles/${name}`, JSON.stringify(requestPayload), {
            headers,
            tags: libTags,
        })
        .json();

    if (response['error']) {
        console.error('Error: createRole', {
            requestPayload,
            response,
        });
    }

    return response;
}

// Create token for existing role.
function createToken(host, headers, roleName) {
    const requestPayload = {
        name: roleName,
        roles: [roleName],
    };

    const response = http
        .post(`${host}/v1/apitokens/generate`, JSON.stringify(requestPayload), {
            headers,
            tags: libTags,
        })
        .json();

    if (response['error']) {
        console.error('Error: createToken', {
            requestPayload,
            response,
        });
    }

    return response;
}

function deleteToken(host, headers, tokenId) {
    http.patch(`${host}/v1/apitokens/revoke/${tokenId}`, '', { headers, tags: libTags });
}

function deleteRole(host, headers, name) {
    http.del(`${host}/v1/roles/${name}`, '', { headers, tags: libTags });
}

function deleteAcessScope(host, headers, accessScopeId) {
    http.del(`${host}/v1/simpleaccessscopes/${accessScopeId}`, '', { headers, tags: libTags });
}

export function createSac(host, headers, roleName, numClusters, numNamespaces) {
    // We want to ensure that randomization used for selecting namespaces is always
    // the same for the same scopes over different executions.
    randomSeed(numClusters * 10000 + numNamespaces);

    const namespacesByClusterMap = getNamespacesByCluster(host, headers);
    const scopeNamespaces = getScopeNamespaces(namespacesByClusterMap, numClusters, numNamespaces);
    const createSacResult = createAccessScopeWithNamespaces(
        host,
        headers,
        roleName,
        scopeNamespaces
    );
    createRole(host, headers, roleName, createSacResult.id);

    const token = createToken(host, headers, roleName);

    return {
        accessScopeId: createSacResult.id,
        token: token['token'],
        tokenId: token['metadata']['id'],
    };
}

export function deleteSac(host, headers, sacInfo, name) {
    deleteToken(host, headers, sacInfo['tokenId']);
    deleteRole(host, headers, name);
    deleteAcessScope(host, headers, sacInfo['accessScopeId']);
}
