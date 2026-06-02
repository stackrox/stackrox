#!/usr/bin/env node

import { createServer } from 'vite';
import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve, relative } from 'node:path';
import { glob } from 'node:fs/promises';

const ROOT = resolve(import.meta.dirname, '../..');
const BODY_PATH = resolve(ROOT, 'src/Containers/MainPage/Body.tsx');
const ROUTE_PATHS_FILE = resolve(ROOT, 'src/routePaths.ts');

// RouteKey → Cypress test globs (relative to ROOT)
// Every RouteKey from Body.tsx MUST have an entry (validated at startup).
// Every test spec in cypress/integration/ MUST be covered by at least one glob (validated at startup).
const routeTestMap = {
    'access-control': ['cypress/integration/accessControl/**', 'cypress/integration/access.test.js'],
    'administration-events': ['cypress/integration/administration/events/**'],
    apidocs: [],
    'apidocs-v2': [],
    'base-images': ['cypress/integration/baseImages/**'],
    clusters: ['cypress/integration/clusters/*.test.*'],
    'clusters/delegated-image-scanning': ['cypress/integration/clusters/delegatedScanning.test.js'],
    'clusters/discovered-clusters': ['cypress/integration/clusters/discovered-clusters/**'],
    'clusters/init-bundles': ['cypress/integration/clusters/init-bundles/**'],
    'clusters/cluster-registration-secrets': ['cypress/integration/clusters/cluster-registration-secrets/**'],
    'clusters/secure-a-cluster': ['cypress/integration/clusters/init-bundles/**'],
    'clusters/secure-a-cluster-crs': ['cypress/integration/clusters/cluster-registration-secrets/**'],
    collections: ['cypress/integration/collections/**'],
    compliance: ['cypress/integration/compliance/**'],
    'compliance-coverage': ['cypress/integration/compliance-enhanced/**'],
    'compliance-schedules': ['cypress/integration/compliance-enhanced/**'],
    configmanagement: ['cypress/integration/configmanagement/**'],
    dashboard: ['cypress/integration/dashboard/**'],
    'exception-configuration': ['cypress/integration/exceptionConfiguration/**'],
    integrations: ['cypress/integration/integrations/**'],
    'listening-endpoints': ['cypress/integration/listeningEndpoints/**'],
    'network-graph': ['cypress/integration/networkGraph/**'],
    'policy-management': ['cypress/integration/policies/**'],
    risk: ['cypress/integration/risk/**'],
    search: [],
    'system-health': ['cypress/integration/systemHealth/**'],
    systemconfig: ['cypress/integration/systemConfig/**', 'cypress/integration/auth.test.js'],
    user: ['cypress/integration/userinfo.test.js'],
    violations: ['cypress/integration/violations/**'],
    'vulnerabilities/exception-management': [
        'cypress/integration/vulnerabilities/exceptionManagement/**',
        'cypress/integration/vulnerabilities/snoozeWorkflow.test.ts',
    ],
    'vulnerabilities/node-cves': ['cypress/integration/vulnerabilities/nodeCves/**'],
    'vulnerabilities/platform-cves': ['cypress/integration/vulnerabilities/platformCves/**'],
    'vulnerabilities/user-workloads': ['cypress/integration/vulnerabilities/workloadCves/**'],
    'vulnerabilities/platform': ['cypress/integration/vulnerabilities/workloadCves/**'],
    'vulnerabilities/all-images': ['cypress/integration/vulnerabilities/workloadCves/**'],
    'vulnerabilities/inactive-images': ['cypress/integration/vulnerabilities/workloadCves/**'],
    'vulnerabilities/images-without-cves': ['cypress/integration/vulnerabilities/workloadCves/**'],
    'vulnerabilities/virtual-machine-cves': ['cypress/integration/vulnerabilities/virtualMachineCves/**'],
    'vulnerabilities/reports': ['cypress/integration/vulnerabilities/VulnerabilityReporting/**'],
    'vulnerability-management': ['cypress/integration/vulnmanagement/**'],
};

// Tests that exercise cross-cutting concerns (login, logout, branding, shell UI).
// These aren't tied to a route — they run when runAll is triggered or when their
// files are directly changed.
const sharedTestSpecs = [
    'cypress/integration/credentialExpiry/credentialExpiry.test.js',
    'cypress/integration/logout.test.js',
    'cypress/integration/main/HelpMenu.test.js',
    'cypress/integration/productBranding.test.js',
    'cypress/integration/telemetry.test.ts',
];

async function validateMappingCompleteness(routeKeys) {
    const errors = [];

    // 1. Every RouteKey from Body.tsx must exist in routeTestMap
    for (const key of routeKeys) {
        if (!(key in routeTestMap)) {
            errors.push(`Route key "${key}" from Body.tsx has no entry in routeTestMap`);
        }
    }

    // 2. Every routeTestMap key must be a valid RouteKey from Body.tsx
    for (const key of Object.keys(routeTestMap)) {
        if (!routeKeys.includes(key)) {
            errors.push(`routeTestMap key "${key}" does not exist in Body.tsx`);
        }
    }

    // 3. Every test spec must be covered by at least one glob or sharedTestSpecs
    const allTestSpecs = [];
    for await (const file of glob(resolve(ROOT, 'cypress/integration/**/*.test.*'))) {
        allTestSpecs.push(relative(ROOT, file));
    }

    const coveredSpecs = new Set();
    const allGlobs = new Set();
    for (const globs of Object.values(routeTestMap)) {
        for (const g of globs) allGlobs.add(g);
    }
    for (const g of allGlobs) {
        for await (const file of glob(resolve(ROOT, g))) {
            coveredSpecs.add(relative(ROOT, file));
        }
    }
    for (const spec of sharedTestSpecs) {
        coveredSpecs.add(spec);
    }

    for (const spec of allTestSpecs) {
        if (!coveredSpecs.has(spec)) {
            errors.push(`Test spec "${spec}" is not covered by any routeTestMap glob or sharedTestSpecs`);
        }
    }

    if (errors.length > 0) {
        console.error('Route-test mapping validation failed:\n');
        for (const err of errors) {
            console.error(`  - ${err}`);
        }
        console.error(
            '\nUpdate routeTestMap or sharedTestSpecs in scripts/ci/ui-affected-tests.mjs'
        );
        process.exit(1);
    }
}

// Parse Body.tsx to extract { routeKey, importPath, pathConstant } entries
function parseRouteEntryPoints() {
    const body = readFileSync(BODY_PATH, 'utf-8');
    const entries = [];

    // Match routeComponentMap entries with both quoted and unquoted keys
    // e.g., 'access-control': { component: asyncComponent(() => import('...')), ... }
    //        clusters: { component: asyncComponent(() => import('...')), ... }
    const mapRegex = /(?:['"]([^'"]+)['"]|([a-zA-Z]\w*))\s*:\s*\{[^}]*?asyncComponent\(\s*\(\)\s*=>\s*import\(\s*'([^']+)'\s*\)/g;
    let match;
    while ((match = mapRegex.exec(body)) !== null) {
        entries.push({ routeKey: match[1] || match[2], importPath: match[3] });
    }

    // Match makeVulnMgmtUserWorkloadView entries (both quoted and unquoted keys)
    const vulnViewRegex = /(?:['"]([^'"]+)['"]|([a-zA-Z]\w*))\s*:\s*\{[^}]*?makeVulnMgmtUserWorkloadView\(/g;
    while ((match = vulnViewRegex.exec(body)) !== null) {
        entries.push({
            routeKey: match[1] || match[2],
            importPath: 'Containers/Vulnerabilities/WorkloadCves/WorkloadCvesPage',
        });
    }

    return entries;
}

// Build reverse map: routePaths export name → RouteKey
function buildPathConstantToRouteKeyMap() {
    const body = readFileSync(BODY_PATH, 'utf-8');
    const routePaths = readFileSync(ROUTE_PATHS_FILE, 'utf-8');

    // Extract path constant names from Body.tsx routeComponentMap: path: someConstant
    const pathAssignments = {};
    const pathRegex = /(?:['"]([^'"]+)['"]|([a-zA-Z]\w*))\s*:\s*\{[^}]*?path:\s*([a-zA-Z]+)/g;
    let match;
    while ((match = pathRegex.exec(body)) !== null) {
        const routeKey = match[1] || match[2];
        pathAssignments[match[3]] = routeKey; // constantName → routeKey
    }

    // Also map routePaths exports to their base path constants
    // e.g., violationsBasePath → 'violations', riskBasePath → 'risk'
    // We need to map ALL exported path constants to a route key.
    // Strategy: for each export in routePaths.ts, check if the constant name
    // is used as a `path:` value in Body.tsx (direct mapping).
    // For constants NOT directly in Body.tsx, find which route they belong to
    // by checking if they share a base path prefix.

    const exportRegex = /export const ([a-zA-Z]+)\s*=/g;
    const allExports = [];
    while ((match = exportRegex.exec(routePaths)) !== null) {
        allExports.push(match[1]);
    }

    // Direct mappings from Body.tsx path: assignments
    const result = { ...pathAssignments };

    // For exports not directly assigned, try to infer route from name
    // e.g., clustersBasePath, clustersPathWithParam → 'clusters'
    // This is a best-effort mapping — false positives are harmless (extra tests)
    for (const exportName of allExports) {
        if (result[exportName]) continue;
        if (exportName === 'mainPath' || exportName === 'isRouteEnabled') continue;

        // Try to match by prefix to an existing route key assignment
        for (const [constName, routeKey] of Object.entries(pathAssignments)) {
            // Check if the export shares a common prefix (e.g., clusters)
            const prefix = constName.replace(/(BasePath|Path|PathWithParam)$/, '');
            const exportPrefix = exportName.replace(/(BasePath|Path|PathWithParam)$/, '');
            if (prefix && exportPrefix && prefix.toLowerCase() === exportPrefix.toLowerCase()) {
                result[exportName] = routeKey;
                break;
            }
        }
    }

    return result;
}

// Crawl Vite module graph from entry point, collecting all source files
async function crawlModuleGraph(server, entrySpecifier, importer) {
    const files = new Set();
    const visited = new Set();

    const resolved = await server.pluginContainer.resolveId(entrySpecifier, importer);
    if (!resolved) return { files, routePathImports: new Set() };

    const rootUrl = '/' + relative(ROOT, resolved.id);
    const routePathImports = new Set();

    async function crawl(url) {
        if (visited.has(url)) return;
        visited.add(url);

        try {
            await server.transformRequest(url);
        } catch {
            return;
        }

        const mod = await server.moduleGraph.getModuleByUrl(url);
        if (!mod?.file) return;

        files.add(mod.file);

        // Check if this module imports from routePaths
        try {
            const source = readFileSync(mod.file, 'utf-8');
            const routePathImportRegex = /import\s*\{([^}]+)\}\s*from\s*['"](?:\.\.\/)*routePaths['"]/g;
            let importMatch;
            while ((importMatch = routePathImportRegex.exec(source)) !== null) {
                const names = importMatch[1].split(',').map((n) => n.trim().split(/\s+as\s+/)[0].trim());
                for (const name of names) {
                    if (name) routePathImports.add(name);
                }
            }
        } catch {
            // skip unreadable files
        }

        for (const dep of mod.importedModules) {
            if (dep.url && dep.file && !dep.file.includes('node_modules')) {
                await crawl(dep.url);
            }
        }
    }

    await crawl(rootUrl);
    return { files, routePathImports };
}

// Expand glob patterns to actual test file paths
async function expandGlobs(patterns) {
    const files = new Set();
    for (const pattern of patterns) {
        const fullPattern = resolve(ROOT, pattern);
        for await (const file of glob(fullPattern)) {
            if (file.endsWith('.test.js') || file.endsWith('.test.ts') || file.endsWith('.test.tsx')) {
                files.add(relative(ROOT, file));
            }
        }
    }
    return [...files].sort();
}

// Parse CLI args
function parseArgs() {
    const args = process.argv.slice(2);
    const flags = { json: false, verbose: false, diff: null, files: [] };

    for (let i = 0; i < args.length; i++) {
        if (args[i] === '--json') flags.json = true;
        else if (args[i] === '--verbose') flags.verbose = true;
        else if (args[i] === '--diff') flags.diff = args[++i];
        else flags.files.push(args[i]);
    }

    return flags;
}

// Get changed files from git diff or CLI args
function getChangedFiles(flags) {
    let files;
    if (flags.diff) {
        // git diff outputs paths relative to repo root (e.g., ui/apps/platform/src/...)
        // We need to filter to only ui/apps/platform/ files and strip that prefix
        const repoRoot = execSync('git rev-parse --show-toplevel', { encoding: 'utf-8' }).trim();
        const relPrefix = relative(repoRoot, ROOT);
        const output = execSync(`git diff --name-only ${flags.diff}...HEAD`, {
            cwd: repoRoot,
            encoding: 'utf-8',
        });
        files = output
            .trim()
            .split('\n')
            .filter(Boolean)
            .filter((f) => f.startsWith(relPrefix + '/'))
            .map((f) => f.slice(relPrefix.length + 1));
    } else {
        files = flags.files;
    }

    // Normalize to absolute paths
    return files.map((f) => resolve(ROOT, f));
}

async function main() {
    const flags = parseArgs();

    if (!flags.diff && flags.files.length === 0) {
        console.error('Usage: node ui-affected-tests.mjs [--diff <ref>] [--json] [--verbose] [files...]');
        process.exit(1);
    }

    const changedFiles = getChangedFiles(flags);
    const changedSrcFiles = changedFiles.filter((f) => f.startsWith(resolve(ROOT, 'src/')));
    const changedTestFiles = changedFiles.filter((f) => f.startsWith(resolve(ROOT, 'cypress/integration/')));
    const changedConfigFiles = changedFiles.filter(
        (f) =>
            f.endsWith('vite.config.js') ||
            f.endsWith('tsconfig.json') ||
            f.endsWith('cypress.config.js') ||
            f.endsWith('package-lock.json') ||
            f.endsWith('index.html')
    );

    // Fast path: config file changes → run all tests
    if (changedConfigFiles.length > 0) {
        const result = { affectedRoutes: {}, testSpecs: [], runAll: true, reason: `Config file changed: ${relative(ROOT, changedConfigFiles[0])}` };
        outputResult(result, flags);
        return;
    }

    // No src files changed — just test files or non-UI files
    if (changedSrcFiles.length === 0 && changedTestFiles.length === 0) {
        const result = { affectedRoutes: {}, testSpecs: [], runAll: false, reason: 'No UI source or test files changed' };
        outputResult(result, flags);
        return;
    }

    const entries = parseRouteEntryPoints();
    const routeKeys = entries.map((e) => e.routeKey);
    await validateMappingCompleteness(routeKeys);

    const pathConstToRoute = buildPathConstantToRouteKeyMap();

    if (flags.verbose) {
        console.error(`Parsed ${entries.length} route entry points from Body.tsx`);
        console.error(`Mapped ${Object.keys(pathConstToRoute).length} path constants to route keys`);
    }

    // Start Vite server in middleware mode for module graph analysis
    if (flags.verbose) console.error('Starting Vite server...');
    const server = await createServer({
        root: ROOT,
        server: { middlewareMode: true },
        optimizeDeps: { noDiscovery: true, include: [] },
        logLevel: 'silent',
    });

    try {
        // Build dependency graph: routeKey → { files, routePathImports }
        const routeDepGraph = new Map();

        for (const { routeKey, importPath } of entries) {
            if (flags.verbose) console.error(`  Crawling: ${routeKey} (${importPath})`);
            const { files, routePathImports } = await crawlModuleGraph(server, importPath, BODY_PATH);
            routeDepGraph.set(routeKey, { files, routePathImports });
            if (flags.verbose) console.error(`    → ${files.size} files, ${routePathImports.size} routePath imports`);
        }

        // Build route adjacency: routeKey → Set<routeKey> (routes it links to)
        const adjacency = new Map();
        for (const [routeKey, { routePathImports }] of routeDepGraph) {
            const linkedRoutes = new Set();
            for (const importName of routePathImports) {
                const linkedRoute = pathConstToRoute[importName];
                if (linkedRoute && linkedRoute !== routeKey) {
                    linkedRoutes.add(linkedRoute);
                }
            }
            adjacency.set(routeKey, linkedRoutes);
        }

        if (flags.verbose) {
            console.error('\nRoute adjacency (links to):');
            for (const [routeKey, linked] of adjacency) {
                if (linked.size > 0) console.error(`  ${routeKey} → ${[...linked].join(', ')}`);
            }
        }

        // Find directly affected routes
        const directlyAffected = new Map(); // routeKey → [triggering files]
        const changedSrcSet = new Set(changedSrcFiles);

        for (const [routeKey, { files }] of routeDepGraph) {
            const triggers = [];
            for (const file of files) {
                if (changedSrcSet.has(file)) triggers.push(relative(ROOT, file));
            }
            if (triggers.length > 0) directlyAffected.set(routeKey, triggers);
        }

        // Check for src files not in any route's tree
        const allRouteFiles = new Set();
        for (const { files } of routeDepGraph.values()) {
            for (const f of files) allRouteFiles.add(f);
        }
        const unmatchedSrcFiles = changedSrcFiles.filter((f) => !allRouteFiles.has(f));

        if (unmatchedSrcFiles.length > 0) {
            const result = {
                affectedRoutes: Object.fromEntries(directlyAffected),
                testSpecs: [],
                runAll: true,
                reason: `Shared/layout file not in any route tree: ${relative(ROOT, unmatchedSrcFiles[0])}`,
            };
            outputResult(result, flags);
            return;
        }

        // Expand via adjacency: if route B is directly affected, also affect routes that link TO B
        const allAffected = new Map(directlyAffected);
        for (const [routeKey, linkedRoutes] of adjacency) {
            for (const linkedRoute of linkedRoutes) {
                if (directlyAffected.has(linkedRoute) && !allAffected.has(routeKey)) {
                    allAffected.set(routeKey, [`(links to ${linkedRoute})`]);
                }
            }
        }

        // Resolve test specs
        const allTestGlobs = new Set();
        for (const routeKey of allAffected.keys()) {
            const globs = routeTestMap[routeKey] || [];
            for (const g of globs) allTestGlobs.add(g);
        }

        // Also include directly changed test files
        const directTestFiles = changedTestFiles
            .filter((f) => f.endsWith('.test.js') || f.endsWith('.test.ts') || f.endsWith('.test.tsx'))
            .map((f) => relative(ROOT, f));

        const expandedSpecs = await expandGlobs([...allTestGlobs]);
        const allSpecs = [...new Set([...expandedSpecs, ...directTestFiles])].sort();

        const result = {
            affectedRoutes: Object.fromEntries(allAffected),
            testSpecs: allSpecs,
            runAll: false,
            reason: null,
        };
        outputResult(result, flags);
    } finally {
        await server.close();
    }
}

function outputResult(result, flags) {
    if (flags.json) {
        console.log(JSON.stringify(result, null, 2));
    } else if (result.runAll) {
        console.log('RUN_ALL');
        if (result.reason) console.error(`Reason: ${result.reason}`);
    } else if (result.testSpecs.length === 0) {
        console.error('No affected test specs found.');
    } else {
        if (flags.verbose) {
            console.error('\nAffected routes:');
            for (const [route, triggers] of Object.entries(result.affectedRoutes)) {
                console.error(`  ${route}: ${triggers.join(', ')}`);
            }
            console.error('');
        }
        for (const spec of result.testSpecs) {
            console.log(spec);
        }
    }
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
