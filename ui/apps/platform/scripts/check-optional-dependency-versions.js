const fs = require('fs');
const path = require('path');

const projectPkgPath = path.resolve('package.json');
const viteRollupPkgPath = path.resolve('node_modules/vite/node_modules/rollup/package.json');
const esbuildPkgPath = path.resolve('node_modules/esbuild/package.json');

const projectPkg = JSON.parse(fs.readFileSync(projectPkgPath, 'utf8'));
const mismatches = [];

function checkBinaries(kind, parentPkgPath, filterPrefix) {
    if (!fs.existsSync(parentPkgPath)) {
        console.warn(`[check-${kind}-binaries] Could not find ${kind} package. Skipping check.`);
        return;
    }

    const parentPkg = JSON.parse(fs.readFileSync(parentPkgPath, 'utf8'));
    const expectedBinaries = parentPkg.optionalDependencies || {};
    const projectBinaries = Object.entries(projectPkg.optionalDependencies || {}).filter(([name]) =>
        name.startsWith(filterPrefix)
    );

    for (const [pkg, declaredVersion] of projectBinaries) {
        const expectedVersion = expectedBinaries[pkg];
        if (!expectedVersion) {
            mismatches.push({
                kind,
                pkg,
                declared: declaredVersion,
                expected: '(missing from upstream optionalDependencies)',
            });
        } else {
            const normalizedDeclared = declaredVersion.replace(/^[^\d]*/, '');
            const normalizedExpected = expectedVersion.replace(/^[^\d]*/, '');
            if (normalizedDeclared !== normalizedExpected) {
                mismatches.push({
                    kind,
                    pkg,
                    declared: declaredVersion,
                    expected: expectedVersion,
                });
            }
        }
    }
}

checkBinaries('rollup', viteRollupPkgPath, '@rollup/rollup-');
checkBinaries('esbuild', esbuildPkgPath, '@esbuild/');

if (mismatches.length > 0) {
    console.warn('\n⚠️ Detected version mismatches for platform-specific binary packages:');
    mismatches.forEach(({ kind, pkg, declared, expected }) => {
        console.warn(`  - (${kind}) ${pkg} is pinned to ${declared}, but wants ${expected}`);
    });
    console.warn(
        '\n👉 To fix: Update the optionalDependencies in `package.json` and run `npm install`'
    );
} else {
    console.log('✅ All Rollup and Esbuild platform binaries match their expected versions.');
}
